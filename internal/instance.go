////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package internal

// instance.go contains the logic for the internal.Instance object along with
// constructors and its methods

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/crypto/nike"
	"gitlab.com/elixxir/crypto/nike/ecdh"
	gpumaths "gitlab.com/elixxir/gpumathsgo"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/internal/state"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/elixxir/server/storage"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/comms/signature"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"gitlab.com/xx_network/primitives/utils"
)

type RoundErrBroadcastFunc func(host *connect.Host, message *mixmessages.RoundError) (*messages.Ack, error)

type EarliestRound struct {
	ClientRoundId    uint64
	GwRoundId        uint64
	GwRoundTimestamp int64
}

// Instance holds long-lived server state
type Instance struct {
	Online            bool
	definition        *Definition
	roundManager      *round.Manager
	resourceQueue     *ResourceQueue
	network           *node.Comms
	streamPool        *gpumaths.StreamPool
	machine           state.Machine
	phaseStateMachine state.GenericMachine

	// RAM storage of rotating node secrets
	nodeSecretManager *storage.NodeSecretManager

	// RAM storage of precanned IDs and keys
	precanStore *storage.PrecanStore

	consensus *network.Instance
	// Denotes that gateway is ready for repeated polling
	isGatewayReady *uint32
	// Denotes that the gateway has successfully contacted its node
	// for the first time
	gatewayFirstPoll *FirstTime

	// Channels
	createRoundQueue   round.Queue
	killInstance       chan chan struct{}
	realtimeRoundQueue round.Queue
	clientErrors       *round.ClientReport

	// Persistent storage object
	storage *storage.Storage

	gatewayPoll          *FirstTime
	requestNewBatchQueue round.Queue

	roundErrFunc RoundErrBroadcastFunc

	errLck         sync.Mutex
	roundError     *mixmessages.RoundError
	recoveredError *mixmessages.RoundError

	phaseOverrides map[int]phase.Phase
	overrideRound  int
	panicWrapper   func(s string)

	gatewayAddress string
	gatewayVersion string
	gatewayMutex   sync.RWMutex

	serverVersion string

	//this is set to 1 if this run the node registered
	firstRun *uint32
	//This is set to 1 after the node has polled for the first time
	firstPoll *uint32

	// Map containing completed batches to pass back to gateway
	completedBatch    map[id.Round]*round.CompletedRound
	completedBatchMux sync.RWMutex

	earliestRoundTracker atomic.Value

	// hostsRegistered tracks whether hosts have been registered from the NDF (0 = not yet, 1 = done)
	hostsRegistered uint32
}

// CreateServerInstance creates a server instance. To actually kick off the server,
// call RunFirstNode() on the resulting ServerInstance.
// After the network object is created, you still need to use it to connect
// to other servers in the network
// Additionally, to clean up the network object (especially in tests), call
// Shutdown() on the network object.
func CreateServerInstance(def *Definition, makeImplementation func(*Instance) *node.Implementation,
	machine state.Machine, version string) (*Instance, error) {
	var err error

	isGwReady := uint32(0)
	firstRun := uint32(0)
	firstPoll := uint32(0)
	instance := &Instance{
		Online:               false,
		definition:           def,
		roundManager:         round.NewManager(),
		resourceQueue:        initQueue(),
		machine:              machine,
		isGatewayReady:       &isGwReady,
		requestNewBatchQueue: round.NewQueue(),
		createRoundQueue:     round.NewQueue(),
		realtimeRoundQueue:   round.NewQueue(),
		killInstance:         make(chan chan struct{}, 1),
		gatewayPoll:          NewFirstTime(),
		completedBatch:       make(map[id.Round]*round.CompletedRound),
		roundError:           nil,
		panicWrapper: func(s string) {
			jww.FATAL.Panic(s)
		},
		serverVersion:        version,
		firstRun:             &firstRun,
		firstPoll:            &firstPoll,
		gatewayFirstPoll:     NewFirstTime(),
		clientErrors:         round.NewClientFailureReport(def.ID),
		phaseStateMachine:    state.NewGenericMachine(),
		earliestRoundTracker: atomic.Value{},
	}

	instance.storage, err = storage.NewStorage(
		def.DbUsername, def.DbPassword, def.DbName,
		def.DbAddress, def.DbPort, def.DevMode)
	if err != nil {
		eMsg := fmt.Sprintf("Could not initialize database: psql://%s@%s:%s/%s: %v",
			def.DbUsername, def.DbAddress, def.DbPort, def.DbName, err)

		if def.DevMode {
			jww.WARN.Printf(eMsg)
		} else {
			jww.FATAL.Panicf(eMsg)
		}
	}

	// Pre-create cache directory to fail fast if there are permission issues
	cacheDir := def.CacheDir
	jww.INFO.Printf("Pre-creating cache directory: %s", cacheDir)
	err = os.MkdirAll(cacheDir, 0755)
	if err != nil {
		jww.WARN.Printf("Failed to pre-create cache directory (cache will be disabled): %+v", err)
		// Continue without caching - non-fatal
	} else {
		jww.INFO.Printf("Cache directory ready: %s", cacheDir)
	}

	// Create node secret manager
	instance.nodeSecretManager = storage.NewNodeSecretManager()

	ephPriv, ephPub := ecdh.ECDHNIKE.NewKeypair(instance.GetRngStreamGen().GetStream())
	instance.nodeSecretManager.UpsertEphemerals(ephPub, ephPriv)

	// Create hardcoded node secret
	// todo: refactor this once a mechanism is implemented for
	//  creating and rotating node secrets.
	h, err := hash.NewCMixHash()
	if err != nil {
		return nil, err
	}

	h.Write(instance.definition.TlsKey)
	nodeSecret := h.Sum(nil)

	err = instance.nodeSecretManager.UpsertSecret(0, nodeSecret)
	if err != nil {
		return nil, errors.Errorf("Could not insert into node secret manager: %v", err)
	}

	// Create stream pool if instructed to use GPU
	if def.UseGPU {
		// Try to initialize the GPU
		// GPU memory allocated in bytes (the same amount is allocated on the CPU side)
		memSize := 200000
		jww.INFO.Printf("Initializing GPU maths, CUDA backend, with memory size %v", memSize)
		var err error
		// It could be better to configure the amount of memory used in a configuration file instead
		instance.streamPool, err = gpumaths.NewStreamPool(2, memSize)
		// An instance without a stream pool is still valid
		// Always panic when we can't do what was intended with the GPU
		if err != nil {
			jww.FATAL.Panicf("Couldn't initialize GPU. Error: %v",
				err.Error())
		}
	} else {
		jww.INFO.Printf("Using CPU maths, rather than CUDA")
	}

	// Initializes the network on this server instance

	//Start local node
	instance.network = node.StartNode(instance.definition.ID, instance.definition.ListeningAddress,
		instance.definition.InterconnectPort, makeImplementation(instance),
		instance.definition.TlsCert, instance.definition.TlsKey)
	instance.roundErrFunc = instance.network.SendRoundError

	// Try to load cached NDFs before creating network instance
	var cachedFullNdf, cachedPartialNdf *ndf.NetworkDefinition
	var fullNdfData, partialNdfData []byte

	jww.INFO.Printf("Attempting to load cached NDFs")

	// Cache paths
	fullNdfCachePath := filepath.Join(def.CacheDir, "full_ndf.json")
	partialNdfCachePath := filepath.Join(def.CacheDir, "partial_ndf.json")

	// Attempt to load full NDF from cache
	fullNdfData, err = storage.LoadNdfFromCache(fullNdfCachePath)
	if err == nil && len(fullNdfData) > 0 {
		jww.DEBUG.Printf("Unmarshaling full NDF from cache (%d bytes)", len(fullNdfData))
		cachedFullNdf, err = ndf.Unmarshal(fullNdfData)
		if err != nil {
			jww.WARN.Printf("Failed to unmarshal cached full NDF, will download fresh: %+v", err)
			cachedFullNdf = nil
			fullNdfData = nil
		} else {
			jww.INFO.Printf("Successfully loaded and unmarshaled full NDF from cache")
		}
	} else if err != nil {
		jww.DEBUG.Printf("No cached full NDF available: %+v", err)
	}

	// Attempt to load partial NDF from cache
	partialNdfData, err = storage.LoadNdfFromCache(partialNdfCachePath)
	if err == nil && len(partialNdfData) > 0 {
		jww.DEBUG.Printf("Unmarshaling partial NDF from cache (%d bytes)", len(partialNdfData))
		cachedPartialNdf, err = ndf.Unmarshal(partialNdfData)
		if err != nil {
			jww.WARN.Printf("Failed to unmarshal cached partial NDF, will download fresh: %+v", err)
			cachedPartialNdf = nil
			partialNdfData = nil
		} else {
			jww.INFO.Printf("Successfully loaded and unmarshaled partial NDF from cache")
		}
	} else if err != nil {
		jww.DEBUG.Printf("No cached partial NDF available: %+v", err)
	}

	// Use cached NDFs if available, otherwise fall back to definition NDFs
	fullNdfToUse := def.FullNDF
	if cachedFullNdf != nil {
		fullNdfToUse = cachedFullNdf
		// Log the hash of the cached full NDF (computed from raw bytes)
		h, _ := hash.NewCMixHash()
		h.Write(fullNdfData)
		fullHash := h.Sum(nil)
		jww.INFO.Printf("Cached full NDF hash: %s",
			base64.StdEncoding.EncodeToString(fullHash))
		jww.INFO.Printf("Using cached full NDF for network instance")
	} else {
		jww.DEBUG.Printf("Using definition full NDF for network instance")
	}

	partialNdfToUse := def.PartialNDF
	if cachedPartialNdf != nil {
		partialNdfToUse = cachedPartialNdf
		// Log the hash of the cached partial NDF (computed from raw bytes)
		h, _ := hash.NewCMixHash()
		h.Write(partialNdfData)
		partialHash := h.Sum(nil)
		jww.INFO.Printf("Cached partial NDF hash: %s",
			base64.StdEncoding.EncodeToString(partialHash))
		jww.INFO.Printf("Using cached partial NDF for network instance")
	} else {
		jww.DEBUG.Printf("Using definition partial NDF for network instance")
	}

	// Initializes the network state tracking on this server instance
	instance.consensus, err = network.NewInstance(instance.network.ProtoComms,
		partialNdfToUse, fullNdfToUse, nil, network.Strict, false)
	if err != nil {
		return nil, errors.WithMessage(err, "Could not initialize network instance")
	}

	// LAZY HOST REGISTRATION FIX:
	// DO NOT register hosts from cached NDF here at startup.
	// Host registration will happen AFTER the first successful connectivity check
	// in permissioning/poll.go. This prevents network saturation during vetting
	// that can cause checkConnectivity to timeout.
	//
	// The hosts will be registered from the current NDF (whether cached or freshly
	// downloaded) after the first poll's connectivity verification passes.
	if cachedFullNdf != nil {
		jww.INFO.Printf("Loaded cached full NDF (%d nodes) - host registration deferred until after connectivity check",
			len(cachedFullNdf.Nodes))
	}

	// Populate hash fields for cached NDFs to avoid unnecessary re-downloads
	if cachedFullNdf != nil && len(fullNdfData) > 0 {
		err = instance.consensus.SetFullNdfHashFromBytes(fullNdfData)
		if err != nil {
			jww.WARN.Printf("Failed to set full NDF hash from cached bytes: %+v", err)
		} else {
			jww.DEBUG.Printf("Successfully populated full NDF hash from cached bytes")
		}
	}

	if cachedPartialNdf != nil && len(partialNdfData) > 0 {
		err = instance.consensus.SetPartialNdfHashFromBytes(partialNdfData)
		if err != nil {
			jww.WARN.Printf("Failed to set partial NDF hash from cached bytes: %+v", err)
		} else {
			jww.DEBUG.Printf("Successfully populated partial NDF hash from cached bytes")
		}
	}

	// NOTE ON NDF HASH INITIALIZATION:
	// The hash fields are now properly populated when loading from cache, which prevents
	// unnecessary re-downloads on the first poll after restart. Previously, these fields
	// remained uninitialized (zeros) because:
	// 1. The wrapper is created with ds.NewNdf() which calls GenerateNDFHash(nil), returning zeros
	// 2. We cannot call Update() to set the hash because it requires RSA signature verification
	// 3. We don't have the permissioning public key available at startup
	//
	// This means the first poll after restart will see hash mismatch (zeros vs actual) and
	// will re-download the NDF from permissioning. However, this one-time download cost is
	// acceptable because:
	// - The downloaded NDF will be cached and verified with proper hash
	// - All subsequent polls will correctly use the cache without re-downloading
	// - This is still vastly better than downloading on every poll (which was the original problem)
	//
	// The hash will be properly set after the first poll's UpdateFullNdf/UpdatePartialNdf
	// calls in permissioning.UpdateNDf(), and then cached for the next restart.

	// Handle overriding local IP
	if instance.GetDefinition().OverrideInternalIP != "" {

		instance.consensus.GetIpOverrideList().Override(instance.GetDefinition().
			ID, instance.GetDefinition().OverrideInternalIP)
	}

	// Connect to our gateway
	_, err = instance.network.AddHost(&id.TempGateway,
		"", instance.definition.Gateway.TlsCert, connect.GetDefaultHostParams())
	if err != nil {
		errMsg := fmt.Sprintf("Count not add dummy gateway %s as host: %+v",
			instance.definition.Gateway.ID, err)
		return nil, errors.New(errMsg)
	}
	_, err = instance.network.AddHost(instance.GetGateway(),
		"", instance.definition.Gateway.TlsCert, connect.GetDefaultHostParams())
	if err != nil {
		errMsg := fmt.Sprintf("Count not add real gateway %s as host: %+v",
			instance.definition.Gateway.ID, err)
		return nil, errors.New(errMsg)
	}

	jww.INFO.Printf("Network Interface Initialized for Node ")

	return instance, nil
}

// RecoverInstance wraps CreateServerInstance, taking a recovered error file
func RecoverInstance(def *Definition, makeImplementation func(*Instance) *node.Implementation,
	machine state.Machine, version string) (*Instance, error) {
	// Create the server instance with normal constructor
	var i *Instance
	var err error
	i, err = CreateServerInstance(def, makeImplementation, machine, version)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to create server instance")
	}

	recoveredErrorEncoded, err := utils.ReadFile(i.definition.RecoveredErrorPath)
	if err != nil {
		return nil, errors.WithMessage(err,
			"Failed to open recovered error file")
	}

	recoveredError, err := base64.StdEncoding.DecodeString(string(recoveredErrorEncoded))
	if err != nil {
		return nil, errors.WithMessagef(err,
			"Failed to base64 decode recovered error file: %s", string(recoveredErrorEncoded))
	}

	// Unmarshal bytes to RoundError
	msg := &mixmessages.RoundError{}
	err = proto.Unmarshal(recoveredError, msg)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to unmarshal message from file")
	}

	// Remove recovered error file
	err = os.Remove(i.definition.RecoveredErrorPath)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to remove ")
	}

	jww.INFO.Printf("Server instance was recovered from error %+v: removing"+
		" file at %s", msg, i.definition.RecoveredErrorPath)

	i.errLck.Lock()
	defer i.errLck.Unlock()
	i.recoveredError = msg

	return i, nil
}

// Run starts the resource queue
func (i *Instance) Run() error {
	go i.resourceQueue.run(i)
	return i.machine.Start()
}

// Shutdown releases any reserved GPU resources
func (i *Instance) Shutdown() {
	if i.streamPool != nil {
		err := i.streamPool.Destroy()
		if err != nil {
			return
		}
	}
}

// GetDefinition returns the server.Definition object
func (i *Instance) GetDefinition() *Definition {
	return i.definition
}

// GetNetworkStatus returns the consensus object
func (i *Instance) GetNetworkStatus() *network.Instance {
	return i.consensus
}

// GetStateMachine returns the round tracking state machine
func (i *Instance) GetStateMachine() state.Machine {
	return i.machine
}

// GetPhaseShareMachine returns state machine tracking the phase share status
// todo: consider removing, may not be needed for final phase share design
func (i *Instance) GetPhaseShareMachine() state.GenericMachine {
	return i.phaseStateMachine
}

// GetGateway returns the id of the node's gateway
func (i *Instance) GetGateway() *id.ID {
	return i.definition.Gateway.ID
}

func (i *Instance) GetSecretManager() *storage.NodeSecretManager {
	return i.nodeSecretManager
}

func (i *Instance) GetPrecanStore() *storage.PrecanStore {
	return i.precanStore
}

func (i *Instance) PopulateDummyUsers(allprecann bool, grp *cyclic.Group) {
	i.precanStore = storage.NewPrecanStore(allprecann, grp)
}

func (i *Instance) AddDummyUserTesting(userId *id.ID, key []byte, grp *cyclic.Group, face interface{}) {
	if i.precanStore == nil {
		i.precanStore = storage.NewPrecanStore(true, grp)
	}
	i.precanStore.AddTesting(userId, key, face)
}

func (i *Instance) SetPrecanStoreTesting(grp *cyclic.Group, face interface{}) {
	switch face.(type) {
	case *testing.T, *testing.M, *testing.B, *testing.PB:
		break
	default:
		jww.FATAL.Panicf("SetSecretManagerTesting is restricted to testing only. Got %T", face)
	}

	i.precanStore = storage.NewPrecanStore(true, grp)

}

func (i *Instance) SetSecretManagerTesting(face interface{}, manager *storage.NodeSecretManager) {
	switch face.(type) {
	case *testing.T, *testing.M, *testing.B, *testing.PB:
		break
	default:
		jww.FATAL.Panicf("SetSecretManagerTesting is restricted to testing only. Got %T", face)
	}

	i.nodeSecretManager = manager
}

// GetRoundManager returns the round manager
func (i *Instance) GetRoundManager() *round.Manager {
	return i.roundManager
}

// GetResourceQueue returns the resource queue used by the server
func (i *Instance) GetResourceQueue() *ResourceQueue {
	return i.resourceQueue
}

// GetGatewayFirstPoll returns the structure which denotes if the node has been fully polled by the gateway
func (i *Instance) GetGatewayFirstPoll() *FirstTime {
	return i.gatewayPoll
}

// GetGatewayFirstContact returns the structure which denotes if the node has been contacted by the gateway
func (i *Instance) GetGatewayFirstContact() *FirstTime {
	return i.gatewayFirstPoll
}

// GetNetwork returns the network object
func (i *Instance) GetNetwork() *node.Comms {
	return i.network
}

// GetID returns this node's ID
func (i *Instance) GetID() *id.ID {
	return i.definition.ID
}

// GetPubKey returns the server DSA public key
func (i *Instance) GetPubKey() *rsa.PublicKey {
	return i.definition.PublicKey
}

// GetPrivKey returns the server RSA private key
func (i *Instance) GetPrivKey() *rsa.PrivateKey {
	return i.definition.PrivateKey
}

// IsFirstRun Sets that this is the first time the node has run
func (i *Instance) IsFirstRun() {
	atomic.StoreUint32(i.firstRun, 1)
}

// GetFirstRun Gets if this is the first time the node has run
func (i *Instance) GetFirstRun() bool {
	return atomic.LoadUint32(i.firstRun) == 1
}

// GetKeepBuffers returns if buffers are to be held on it
func (i *Instance) GetKeepBuffers() bool {
	return i.definition.Flags.KeepBuffers
}

// GetRegServerPubKey returns the public key of the registration server
func (i *Instance) GetRegServerPubKey() *rsa.PublicKey {
	return i.definition.Network.PublicKey
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

// IsFirstPoll returns true if this is the
// first time this is called, otherwise returns false
func (i *Instance) IsFirstPoll() bool {
	return atomic.SwapUint32(i.firstPoll, 1) == 0
}

// GetRngStreamGen returns the fastRNG StreamGenerator in definition.
func (i *Instance) GetRngStreamGen() *fastRNG.StreamGenerator {
	return i.definition.RngStreamGen
}

// GetIP returns the public IP of the node from the instance
func (i *Instance) GetIP() string {
	return i.definition.PublicAddress
}

// GetResourceMonitor returns the resource monitoring object
func (i *Instance) GetResourceMonitor() *measure.ResourceMonitor {
	return i.definition.ResourceMonitor
}

func (i *Instance) GetKillChan() chan chan struct{} {
	return i.killInstance
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

func (i *Instance) GetClientReport() *round.ClientReport {
	return i.clientErrors
}

func (i *Instance) GetRoundError() *mixmessages.RoundError {
	return i.roundError
}

func (i *Instance) GetRecoveredError() *mixmessages.RoundError {
	i.errLck.Lock()
	defer i.errLck.Unlock()
	return i.recoveredError
}

// only use if you already have the error lock
// TODO - find a way to remove
func (i *Instance) GetRecoveredErrorUnsafe() *mixmessages.RoundError {
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

func (i *Instance) SendRoundError(h *connect.Host, m *mixmessages.RoundError) (*messages.Ack, error) {
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
	jww.TRACE.Printf("Returning Gateway: %s, %s", i.gatewayAddress,
		i.gatewayVersion)
	return i.gatewayAddress, i.gatewayVersion
}

// UpsertGatewayData saves the gateway address and version to the instance, if
// they differ. Panics if the gateway address is empty.
func (i *Instance) UpsertGatewayData(addr string, ver string) {
	jww.TRACE.Printf("Upserting Gateway: %s, %s", addr, ver)

	if addr == "" {
		jww.FATAL.Panicf("Faild to upsert gateway data, gateway address is empty.")
	}

	i.gatewayMutex.Lock()
	defer i.gatewayMutex.Unlock()

	if i.gatewayAddress != addr || i.gatewayVersion != ver {
		(*i).gatewayAddress = addr
		(*i).gatewayVersion = ver
	}
}

/* TESTING FUNCTIONS */
func (i *Instance) OverridePhases(overrides map[int]phase.Phase) {
	i.phaseOverrides = overrides
	i.overrideRound = -1
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
	case *testing.T, *testing.M, *testing.B, *testing.PB:
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

// reports an error from the node which is not associated with a round
func (i *Instance) ReportNodeFailure(errIn error) {
	i.ReportRoundFailure(errIn, i.GetID(), 0)
}

// reports an error from a different node in the round the node is participating in
func (i *Instance) ReportRemoteFailure(roundErr *mixmessages.RoundError) {
	i.reportFailure(roundErr)
}

// Create a round error, pass the error over the channel and update the state to
// ERROR state. In situations that cause critical panic level errors.
func (i *Instance) ReportRoundFailure(errIn error, nodeId *id.ID, roundId id.Round) {

	//truncate the error if it is too long
	errStr := errIn.Error()
	if len(errStr) > 5000 {
		errStr = errStr[:5000]
	}

	roundErr := mixmessages.RoundError{
		Id:     uint64(roundId),
		Error:  errStr,
		NodeId: nodeId.Marshal(),
	}

	i.reportFailure(&roundErr)
}

// Create a round error, pass the error over the channel and update the state to
// ERROR state. In situations that cause critical panic level errors.
func (i *Instance) reportFailure(roundErr *mixmessages.RoundError) {
	i.errLck.Lock()
	defer i.errLck.Unlock()

	nodeId, _ := id.Unmarshal(roundErr.NodeId)

	//sign the round error
	err := signature.SignRsa(roundErr, i.GetPrivKey())
	if err != nil {
		jww.FATAL.Panicf("Failed to sign round state update of: %s "+
			"\n roundError: %+v", err, roundErr)
	}

	//then call update state err
	sm := i.GetStateMachine()

	currentActivity := sm.Get()
	// TODO In the future, we should write code to clean up an in-progress round
	//  that has an error. In that case, we should also reevaluate this logic,
	//  as it probably won't work as intended anymore.
	if currentActivity == current.ERROR || currentActivity == current.CRASH {
		// There's already an error, so there's no need to change to error state
		jww.FATAL.Printf("Round failure reported, but the node is already in ERROR state. RoundID %v; nodeID %v; error text %v",
			roundErr.Id, nodeId, roundErr.Error)
		return
	}

	// put the new error in the instance, since the node isn't currently in
	// an error or crash state
	i.roundError = roundErr

	// Change instance state to ERROR
	ok, err := sm.Update(current.ERROR)
	if err != nil {
		jww.FATAL.Panicf("Failed to change state to ERROR state: %v", err)
	}
	if !ok {
		jww.FATAL.Panicf("Failed to change state to ERROR state")
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

// WaitUntilRoundCompletes is called once a kill signal is received.
// It returns on one of two conditions: Either the current round is completed,
// or duration time units have occurred, causing a timeout.
// Round completion is monitored by sending a channel through another
// channel (chan chan struct{}), and on round completion,
// we send to that channel and receive here.
func (i *Instance) WaitUntilRoundCompletes(duration time.Duration) {
	k := make(chan struct{})
	jww.INFO.Printf("Waiting for round to complete before closing...")
	i.killInstance <- k
	jww.TRACE.Printf("Sent kill signal, waiting for response")
	select {
	case <-k:
		jww.INFO.Printf("Round completed, closing!\n")
	case <-time.After(duration):
		jww.ERROR.Print("Round took too long to complete, closing!")
	}
}

func (i *Instance) AddCompletedBatch(cr *round.CompletedRound) error {
	i.completedBatchMux.Lock()
	defer i.completedBatchMux.Unlock()
	i.completedBatch[cr.RoundID] = cr
	return nil
}

func (i *Instance) GetCompletedBatch(rid id.Round) (*round.CompletedRound, bool) {
	i.completedBatchMux.Lock()

	defer i.completedBatchMux.Unlock()
	cr, ok := i.completedBatch[rid]
	delete(i.completedBatch, rid)
	return cr, ok
}

const NoCompletedBatch = "No round to report on"

func (i *Instance) GetCompletedBatchRID() (id.Round, error) {
	i.completedBatchMux.RLock()
	defer i.completedBatchMux.RUnlock()

	for roundId := range i.completedBatch {
		return roundId, nil
	}

	return 0, errors.New(NoCompletedBatch)
}

func (i *Instance) GetEarliestRound() (uint64, uint64, int64, error) {
	earliestRound, ok := i.earliestRoundTracker.Load().(EarliestRound)
	if !ok {
		return 0, 0, 0, errors.New("Earliest round state does not exist, try again")
	}

	return earliestRound.ClientRoundId,
		earliestRound.GwRoundId, earliestRound.GwRoundTimestamp, nil
}

func (i *Instance) SetEarliestRound(newEarliestClientRoundId,
	newEarliestGwRoundId uint64, newGwEarliestTs int64) {
	newEarliestRound := EarliestRound{
		ClientRoundId:    newEarliestClientRoundId,
		GwRoundId:        newEarliestGwRoundId,
		GwRoundTimestamp: newGwEarliestTs,
	}

	i.earliestRoundTracker.Store(newEarliestRound)
}

func (i *Instance) GetEd25519Key() nike.PublicKey {
	pub, _ := i.nodeSecretManager.GetEphemeralEd()
	return pub
}

// IsHostsRegistered checks if hosts have been registered from the NDF
func (i *Instance) IsHostsRegistered() bool {
	return atomic.LoadUint32(&i.hostsRegistered) == 1
}

// SetHostsRegistered sets the hosts registered flag to true. Returns true if it was previously false.
func (i *Instance) SetHostsRegistered() bool {
	return atomic.CompareAndSwapUint32(&i.hostsRegistered, 0, 1)
}
