////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package internal

// instance.go contains the logic for the internal.Instance object along with
// constructors and it's methods

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/gpumaths"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/internal/state"
	"gitlab.com/elixxir/server/services"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

type RoundErrBroadcastFunc func(host *connect.Host, message *mixmessages.RoundError) (*mixmessages.Ack, error)

// Holds long-lived server state
type Instance struct {
	Online        bool
	definition    *Definition
	roundManager  *round.Manager
	resourceQueue *ResourceQueue
	network       *node.Comms
	streamPool    *gpumaths.StreamPool
	machine       state.Machine

	consensus      *network.Instance
	isGatewayReady *uint32

	// isAfterFirstPoll is a flag indicating that the gateway
	//  has polled successfully for the first time.
	//  It is set atomically.
	isAfterFirstPoll *uint32

	// markFirstPoll is used to set isAfterFirstPoll once and only once
	markFirstPoll sync.Once

	// Channels
	createRoundQueue    round.Queue
	completedBatchQueue round.CompletedQueue
	realtimeRoundQueue  round.Queue

	gatewayPoll          *FirstTime
	requestNewBatchQueue round.Queue

	roundErrFunc RoundErrBroadcastFunc

	errLck                 sync.Mutex
	roundError             *mixmessages.RoundError
	recoveredError         *mixmessages.RoundError
	RecoveredErrorFilePath string

	phaseOverrides map[int]phase.Phase
	overrideRound  int
	panicWrapper   func(s string)

	gatewayAddess  string
	gatewayVersion string
	gatewayMutex   sync.RWMutex

	serverVersion string
}

// Create a server instance. To actually kick off the server,
// call RunFirstNode() on the resulting ServerInstance.
// After the network object is created, you still need to use it to connect
// to other servers in the network
// Additionally, to clean up the network object (especially in tests), call
// Shutdown() on the network object.
func CreateServerInstance(def *Definition, makeImplementation func(*Instance) *node.Implementation,
	machine state.Machine, useGPU bool, version string) (*Instance, error) {
	isGwReady := uint32(0)
	firstPoll := uint32(0)
	instance := &Instance{
		Online:               false,
		definition:           def,
		roundManager:         round.NewManager(),
		resourceQueue:        initQueue(),
		machine:              machine,
		isGatewayReady:       &isGwReady,
		isAfterFirstPoll:     &firstPoll,
		requestNewBatchQueue: round.NewQueue(),
		createRoundQueue:     round.NewQueue(),
		realtimeRoundQueue:   round.NewQueue(),
		completedBatchQueue:  round.NewCompletedQueue(),
		gatewayPoll:          NewFirstTime(),
		roundError:           nil,
		panicWrapper: func(s string) {
			jww.FATAL.Panic(s)
		},
		serverVersion: version,
	}

	// Create stream pool if instructed to use GPU
	if useGPU {
		// Try to initialize the GPU
		// GPU memory allocated in bytes (the same amount is allocated on the CPU side)
		memSize := 268435456
		jww.INFO.Printf("Initializing GPU maths, CUDA backend, with memory size %v", memSize)
		var err error
		// It could be better to configure the amount of memory used in a configuration file instead
		instance.streamPool, err = gpumaths.NewStreamPool(2, memSize)
		// An instance without a stream pool is still valid
		// So, log the error here instead of returning it, because we didn't fail to create the server instance here
		if err != nil {
			jww.ERROR.Printf("Couldn't initialize GPU. Falling back to CPU math. Error: %v", err.Error())
		}
	} else {
		jww.INFO.Printf("Using CPU maths, rather than CUDA")
	}

	// Initializes the network on this server instance

	//Start local node
	instance.network = node.StartNode(instance.definition.ID, instance.definition.Address,
		makeImplementation(instance), instance.definition.TlsCert, instance.definition.TlsKey)
	instance.roundErrFunc = instance.network.SendRoundError

	// Initializes the network state tracking on this server instance
	var err error
	instance.consensus, err = network.NewInstance(instance.network.ProtoComms, def.PartialNDF, def.FullNDF)
	if err != nil {
		return nil, errors.WithMessage(err, "Could not initialize network instance")
	}

	// Connect to our gateway. At this point we should only know our gateway as this should occur
	//  BEFORE polling
	err = instance.GetConsensus().UpdateGatewayConnections()
	if err != nil {
		return nil, errors.Errorf("Could not update gateway connections: %+v", err)
	}

	// Add gateways to host object
	_, err = instance.network.AddHost(&id.TempGateway,
		"", instance.definition.Gateway.TlsCert, false, true)
	if err != nil {
		errMsg := fmt.Sprintf("Count not add gateway %s as host: %+v",
			instance.definition.Gateway.ID, err)
		return nil, errors.New(errMsg)
	}
	jww.INFO.Printf("Network Interface Initilized for Node ")

	return instance, nil
}

// Wrap CreateServerInstance, taking a recovered error file
func RecoverInstance(def *Definition, makeImplementation func(*Instance) *node.Implementation,
	machine state.Machine, useGPU bool, version string, recoveredErrorFile *os.File) (*Instance, error) {
	// Create the server instance with normal constructor
	var i *Instance
	var err error
	i, err = CreateServerInstance(def, makeImplementation, machine, useGPU, version)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to create server instance")
	}
	i.RecoveredErrorFilePath = recoveredErrorFile.Name()

	// Read recovered error file to bytes
	var recoveredError []byte
	_, err = recoveredErrorFile.Read(recoveredError)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to read recovered error")
	}

	// Close recovered error file
	err = recoveredErrorFile.Close()
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to close recovered error file")
	}

	// Remove recovered error file
	err = os.Remove(recoveredErrorFile.Name())
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to remove ")
	}

	// Unmarshal bytes to RoundError, set recoveredError field on instance
	msg := &mixmessages.RoundError{}
	err = proto.Unmarshal(recoveredError, msg)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to unmarshal message from file")
	}

	i.errLck.Lock()
	defer i.errLck.Unlock()
	i.recoveredError = msg

	return i, nil
}

// RestartNetwork is intended to reset the network with newly signed certs obtained from polling
// permissioning
func (i *Instance) RestartNetwork(makeImplementation func(*Instance) *node.Implementation,
	serverCert, gwCert string) error {

	jww.INFO.Printf("Restarting network...")
	// Shut down the network so we can restart
	i.network.Shutdown()

	// Set definition for newly signed certs
	i.definition.TlsCert = []byte(serverCert)
	i.definition.Gateway.TlsCert = []byte(gwCert)

	// Get the id and cert
	ourId := i.GetID()
	ourDef := i.GetDefinition()

	// Reset the network with the newly signed certs
	i.network = node.StartNode(ourId, ourDef.Address,
		makeImplementation(i), ourDef.TlsCert, ourDef.TlsKey)

	// Connect to the Permissioning Server with authentication enabled
	_, err := i.network.AddHost(&id.Permissioning,
		i.definition.Permissioning.Address, i.definition.Permissioning.TlsCert, true, true)
	if err != nil {
		return err
	}

	_, err = i.network.AddHost(i.definition.Gateway.ID, "",
		i.definition.Gateway.TlsCert, false, true)

	i.consensus.SetProtoComms(i.network.ProtoComms)
	err = i.consensus.UpdateNodeConnections()

	return err
}

// Run starts the resource queue
func (i *Instance) Run() error {
	go i.resourceQueue.run(i)
	return i.machine.Start()
}

// GetDefinition returns the server.Definition object
func (i *Instance) GetDefinition() *Definition {
	return i.definition
}

// GetTopology returns the consensus object
func (i *Instance) GetConsensus() *network.Instance {
	return i.consensus
}

// GetStateMachine returns the consensus object
func (i *Instance) GetStateMachine() state.Machine {
	return i.machine
}

// GetGateway returns the id of the node's gateway
func (i *Instance) GetGateway() *id.ID {
	return i.definition.Gateway.ID
}

//GetUserRegistry returns the user registry used by the server
func (i *Instance) GetUserRegistry() globals.UserRegistry {
	return i.definition.UserRegistry
}

//GetRoundManager returns the round manager
func (i *Instance) GetRoundManager() *round.Manager {
	return i.roundManager
}

//GetResourceQueue returns the resource queue used by the server
func (i *Instance) GetResourceQueue() *ResourceQueue {
	return i.resourceQueue
}

// GetNetwork returns the network object
func (i *Instance) GetNetwork() *node.Comms {
	return i.network
}

//GetID returns this node's ID
func (i *Instance) GetID() *id.ID {
	return i.definition.ID
}

//GetPubKey returns the server DSA public key
func (i *Instance) GetPubKey() *rsa.PublicKey {
	return i.definition.PublicKey
}

//GetPrivKey returns the server DSA private key
func (i *Instance) GetPrivKey() *rsa.PrivateKey {
	return i.definition.PrivateKey
}

//GetKeepBuffers returns if buffers are to be held on it
func (i *Instance) GetKeepBuffers() bool {
	return i.definition.Flags.KeepBuffers
}

//GetRegServerPubKey returns the public key of the registration server
func (i *Instance) GetRegServerPubKey() *rsa.PublicKey {
	return i.definition.Permissioning.PublicKey
}

// FIXME Populate this from the YAML or something
func (i *Instance) GetGraphGenerator() services.GraphGenerator {
	return i.definition.GraphGenerator
}

// GetMetricsLog returns the log path for metrics data
func (i *Instance) GetMetricsLog() string {
	return i.definition.MetricLogPath
}

// GetServerCertPath returns the path for Server certificate
func (i *Instance) GetServerCertPath() string {
	return i.definition.ServerCertPath
}

// GetGatewayCertPath returns the path for Gateway certificate
func (i *Instance) GetGatewayCertPath() string {
	return i.definition.GatewayCertPath
}

// GetRngStreamGen returns the fastRNG StreamGenerator in definition.
func (i *Instance) GetRngStreamGen() *fastRNG.StreamGenerator {
	return i.definition.RngStreamGen
}

// GetIP returns the IP of the node from the instance
func (i *Instance) GetIP() string {
	return i.definition.Address
}

// GetResourceMonitor returns the resource monitoring object
func (i *Instance) GetResourceMonitor() *measure.ResourceMonitor {
	return i.definition.ResourceMonitor
}

func (i *Instance) GetGatewayFirstTime() *FirstTime {
	return i.gatewayPoll
}

func (i *Instance) GetCompletedBatchQueue() round.CompletedQueue {
	return i.completedBatchQueue
}

func (i *Instance) GetCreateRoundQueue() round.Queue {
	return i.createRoundQueue
}

func (i *Instance) GetRealtimeRoundQueue() round.Queue {
	return i.realtimeRoundQueue
}

func (i *Instance) GetRequestNewBatchQueue() round.Queue {
	return i.requestNewBatchQueue
}

func (i *Instance) GetRoundError() *mixmessages.RoundError {
	return i.roundError
}

func (i *Instance) GetRecoveredError() *mixmessages.RoundError {
	return i.recoveredError
}

func (i *Instance) GetServerVersion() string {
	return i.serverVersion
}

func (i *Instance) ClearRecoveredError() {
	i.errLck.Lock()
	defer i.errLck.Unlock()
	i.recoveredError = nil
}

func (i *Instance) IsReadyForGateway() bool {
	ourVal := atomic.LoadUint32(i.isGatewayReady)

	return ourVal == 1
}

func (i *Instance) SetGatewayAsReady() {
	atomic.CompareAndSwapUint32(i.isGatewayReady, 0, 1)
}

// IsAfterFirstPoll checks if the isAfterFirstPoll has been set
// The default instance value is 0, indicating gateway
//  has not polled yet
// The set value is 1, indicating gateway has successfully polled
func (i *Instance) IsAfterFirstPoll() bool {
	ourVal := atomic.LoadUint32(i.isAfterFirstPoll)
	return ourVal == 1
}

// DeclareFirstPoll sets the isAfterFirstPoll variable.
//  This uses a sync.Once variable to ensure the variable is only set once
func (i *Instance) DeclareFirstPoll() {
	i.markFirstPoll.Do(func() {
		atomic.CompareAndSwapUint32(i.isAfterFirstPoll, 0, 1)
	})

}

func (i *Instance) SendRoundError(h *connect.Host, m *mixmessages.RoundError) (*mixmessages.Ack, error) {
	jww.FATAL.Printf("Sending round error to %+v\n", h)
	return i.roundErrFunc(h, m)
}

func (i *Instance) GetPhaseOverrides() map[int]phase.Phase {
	return i.phaseOverrides
}

func (i *Instance) GetOverrideRound() int {
	return i.overrideRound
}

func (i *Instance) GetPanicWrapper() func(s string) {
	return i.panicWrapper
}

func (i *Instance) GetGatewayData() (addr string, ver string) {
	i.gatewayMutex.RLock()
	defer i.gatewayMutex.RUnlock()
	return i.gatewayAddess, i.gatewayVersion
}

func (i *Instance) UpsertGatewayData(addr string, ver string) {
	i.gatewayMutex.Lock()
	defer i.gatewayMutex.Unlock()
	if i.gatewayAddess != addr || i.gatewayVersion != ver {
		i.gatewayVersion = addr
		i.gatewayVersion = ver
	}
}

/* TESTING FUNCTIONS */
func (i *Instance) OverridePhases(overrides map[int]phase.Phase) {
	i.phaseOverrides = overrides
}

func (i *Instance) OverridePhasesAtRound(overrides map[int]phase.Phase, round int) {
	i.phaseOverrides = overrides
	i.overrideRound = round
}

func (i *Instance) SetRoundErrFunc(f RoundErrBroadcastFunc, t *testing.T) {
	if t == nil {
		panic("Cannot call this outside of tests")
	}
	i.roundErrFunc = f
}

func (i *Instance) SetTestRecoveredError(m *mixmessages.RoundError, t *testing.T) {
	if t == nil {
		panic("This cannot be used outside of a test")
	}
	i.errLck.Lock()
	defer i.errLck.Unlock()
	i.recoveredError = m
}

func (i *Instance) SetTestRoundError(m *mixmessages.RoundError, t *testing.T) {
	if t == nil {
		panic("This cannot be used outside of a test")
	}
	i.errLck.Lock()
	defer i.errLck.Unlock()
	i.roundError = m
}

func (i *Instance) OverridePanicWrapper(f func(s string), t *testing.T) {
	if t == nil {
		panic("OverridePanicWrapper cannot be used outside of a test")
	}
	i.panicWrapper = f
}

// GenerateId generates a random ID and returns it
// FIXME: This function needs to be replaced
func GenerateId(i interface{}) *id.ID {
	switch i.(type) {
	case *testing.T:
		break
	case *testing.M:
		break
	default:
		jww.FATAL.Panicf("GenerateId is restricted to testing only. Got %T", i)
		return nil
	}

	jww.WARN.Printf("GenerateId needs to be replaced")

	// Create node id buffer
	nodeIdBytes := make([]byte, id.ArrIDLen)
	rng := csprng.NewSystemRNG()

	// Generate random bytes and store in buffer
	_, err := rng.Read(nodeIdBytes)
	if err != nil {
		err = errors.New(err.Error())
		jww.FATAL.Panicf("Could not generate random nodeID: %+v", err)
	}

	nid, err := id.Unmarshal(nodeIdBytes)
	if err != nil {
		err = errors.New(err.Error())
		jww.FATAL.Panicf("Could not unmarshal nodeID: %+v", err)
	}

	return nid
}

// Create a round error, pass the error over the chanel and update the state to ERROR state
// In situations that cause critical panic level errors.
func (i *Instance) ReportRoundFailure(errIn error, nodeId *id.ID, roundId *id.Round) {
	i.errLck.Lock()
	defer i.errLck.Unlock()
	if roundId == nil {
		jww.FATAL.Panicf("Encountered an unrecoverable error: " + errIn.Error())
	}
	roundErr := mixmessages.RoundError{
		Id:     uint64(*roundId),
		Error:  errIn.Error(),
		NodeId: nodeId.Marshal(),
	}
	// pass the error over the chanel
	//instance get err chan
	i.roundError = &roundErr

	//then call update state err
	sm := i.GetStateMachine()
	ok, err := sm.Update(current.ERROR)
	if err != nil {
		log.Panicf("Failed to change state to ERROR STATE %v", err)
	}

	if !ok {
		log.Panicf("Failed to change state to ERROR STATE")
	}
}

func (i *Instance) String() string {
	nid := i.definition.ID
	localServer := i.network.String()
	port := strings.Split(localServer, ":")[1]
	addr := fmt.Sprintf("%s:%s", nid, port)
	return addr
}

func (i *Instance) GetStreamPool() *gpumaths.StreamPool {
	return i.streamPool
}

// GetDisableStreaming returns the DisableStreaming boolean that determines if
// streaming will be used.
func (i *Instance) GetDisableStreaming() bool {
	return i.definition.DisableStreaming
}
