////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package server

import (
	"crypto/x509"
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/crypto/tls"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/server/state"
	"gitlab.com/elixxir/server/services"
	"sync/atomic"
	"testing"
)

// Holds long-lived server state
type Instance struct {
	Online        bool
	definition    *Definition
	roundManager  *round.Manager
	resourceQueue *ResourceQueue
	network       *node.Comms
	machine       state.Machine

	consensus      *network.Instance
	isGatewayReady *uint32
	// Channels
	createRoundQueue    round.Queue
	completedBatchQueue round.CompletedQueue
	realtimeRoundQueue  round.Queue

	gatewayPoll          *FirstTime
	requestNewBatchQueue round.Queue
}

// Create a server instance. To actually kick off the server,
// call RunFirstNode() on the resulting ServerInstance.
// After the network object is created, you still need to use it to connect
// to other servers in the network
// Additionally, to clean up the network object (especially in tests), call
// Shutdown() on the network object.
func CreateServerInstance(def *Definition, makeImplementation func(*Instance) *node.Implementation,
	machine state.Machine, noTls bool) (*Instance, error) {
	isGwReady := uint32(0)

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
		completedBatchQueue:  round.NewCompletedQueue(),
		gatewayPoll:          NewFirstTime(),
	}

	//Start local node
	instance.network = node.StartNode(instance.definition.ID.String(), instance.definition.Address,
		makeImplementation(instance), instance.definition.TlsCert, instance.definition.TlsKey)

	if noTls {
		instance.network.DisableAuth()
	}

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
	if instance.definition.Gateway.Address != "" {
		_, err := instance.network.AddHost(instance.definition.Gateway.ID.String(),
			instance.definition.Gateway.Address, instance.definition.Gateway.TlsCert, false, true)
		if err != nil {
			errMsg := fmt.Sprintf("Count not add gateway %s as host: %+v",
				instance.definition.Gateway.ID, err)
			return nil, errors.New(errMsg)

		}
	} else {
		jww.WARN.Printf("No Gateway avalible, starting without gateway")
	}
	jww.INFO.Printf("Network Interface Initilized for Node ")

	return instance, nil
}

// RestartNetwork is intended to reset the network with newly signed certs obtained from polling
// permissioning
func (i *Instance) RestartNetwork(makeImplementation func(*Instance) *node.Implementation, definition *Definition, noTls bool) error {
	i.network.Shutdown()
	i.definition = definition
	i.network = node.StartNode(definition.ID.String(), definition.Address,
		makeImplementation(i), definition.TlsCert, definition.TlsKey)

	if noTls {
		i.network.DisableAuth()
	}

	// Connect to the Permissioning Server with authentication enabled
	_, err := i.network.AddHost(id.PERMISSIONING,
		i.definition.Permissioning.Address, i.definition.Permissioning.TlsCert, true, true)
	if err != nil {
		return err
	}

	_, err = i.network.AddHost(i.definition.Gateway.ID.String(), i.definition.Gateway.Address,
		i.definition.Gateway.TlsCert, false, true)

	return err
}

// Run starts the resource queue
func (i *Instance) Run() error {
	go i.resourceQueue.run(i)
	return i.machine.Start()
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
func (i *Instance) GetGateway() *id.Gateway {
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
func (i *Instance) GetID() *id.Node {
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

//IsRegistrationAuthenticated returns the skipReg parameter
func (i *Instance) IsRegistrationAuthenticated() bool {
	return i.definition.Flags.SkipReg
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

// GetRngStreamGen returns the fastRNG StreamGenerator in definition.
func (i *Instance) GetRngStreamGen() *fastRNG.StreamGenerator {
	return i.definition.RngStreamGen
}

// GetIP returns the IP of the node from the instance
func (i *Instance) GetIP() string {
	/*fmt.Printf("i.definition.Nodes: %+v\n", i.definition.Nodes)
	fmt.Printf("i.GetTopology(): %+v\n", i.GetTopology())
	fmt.Printf("i.GetID(): %+v\n", i.GetID())
	addrWithPort := i.definition.Nodes[i.GetTopology().GetNodeLocation(i.GetID())].Address
	return strings.Split(addrWithPort, ":")[0]*/
	return ""
}

// GetResourceMonitor returns the resource monitoring object
func (i *Instance) GetResourceMonitor() *measure.ResourceMonitor {
	return i.definition.ResourceMonitor
}

func (i *Instance) GetRoundCreationTimeout() int {
	return i.definition.RoundCreationTimeout
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

// todo: docstring
func (i *Instance) GetRealtimeRoundQueue() round.Queue {
	return i.realtimeRoundQueue
}

func (i *Instance) GetRequestNewBatchQueue() round.Queue {
	return i.requestNewBatchQueue
}

func (i *Instance) IsReadyForGateway() bool {
	ourVal := atomic.LoadUint32(i.isGatewayReady)

	return ourVal == 1
}

func (i *Instance) SetGatewayAsReady() {
	atomic.StoreUint32(i.isGatewayReady, 1)
}

// GenerateId generates a random ID and returns it
// FIXME: This function needs to be replaced
func GenerateId(i interface{}) *id.Node {
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
	nodeIdBytes := make([]byte, id.NodeIdLen)
	rng := csprng.NewSystemRNG()

	// Generate random bytes and store in buffer
	_, err := rng.Read(nodeIdBytes)
	if err != nil {
		err := errors.New(err.Error())
		jww.FATAL.Panicf("Could not generate random nodeID: %+v", err)
	}

	nid := id.NewNodeFromBytes(nodeIdBytes)

	return nid
}

// VerifyTopology checks the signed node certs and verifies that no falsely signed certs are submitted
// it then shuts down the network so that it can be reinitialized with the new topology
func (i *Instance) VerifyTopology() error {
	//Load Permissioning cert into a cert object
	permissioningCert, err := tls.LoadCertificate(string(i.definition.Permissioning.TlsCert))
	if err != nil {
		jww.ERROR.Printf("Could not load the permissioning server cert: %v", err)
		return err
	}

	// FIXME: Force the permissioning cert to act as a CA
	permissioningCert.BasicConstraintsValid = true
	permissioningCert.IsCA = true
	permissioningCert.KeyUsage = x509.KeyUsageCertSign
	/*
		//Iterate through the topology
		for j := 0; j < i.definition.Topology.Len(); j++ {
			//Load the node Cert from topology
			nodeCert, err := tls.LoadCertificate(string(i.definition.Nodes[j].TlsCert))
			if err != nil {
				errorMsg := fmt.Sprintf("Could not load the node %v's certificate cert: %v", j, err)
				return errors.New(errorMsg)
			}

			//Check that the node's cert was signed by the permissioning server's cert
			err = nodeCert.CheckSignatureFrom(permissioningCert)
			if err != nil {
				errorMsg := fmt.Sprintf("Could not verify that a node %v's cert was signed by permissioning: %v", j, err)
				return errors.New(errorMsg)
			}
		}
	*/
	return nil
}

/*
// String adheres to the stringer interface, returns unique identifying
// information about the node
func (i *Instance) String() string {
	nid := i.definition.ID
	localServer := i.network.String()
	port := strings.Split(localServer, ":")[1]
	addr := fmt.Sprintf("%s:%s", nid, port)
	return services.NameStringer(addr, myLoc, numNodes)
}


*/
