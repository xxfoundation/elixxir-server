package server

import (
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/server/conf"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/services"
	"google.golang.org/grpc/credentials"
	"runtime"
)

// Holds long-lived server state
type Instance struct {
	roundManager    *round.Manager
	network         *node.NodeComms
	resourceQueue   *ResourceQueue
	userReg         globals.UserRegistry
	pubKey          *signature.DSAPublicKey
	privKey         *signature.DSAPrivateKey
	regServerPubKey *signature.DSAPublicKey
	topology        *circuit.Circuit
	thisNode        *id.Node
	graphGenerator  services.GraphGenerator
	firstNode
	LastNode
	params *conf.Params
}

func (i *Instance) GetTopology() *circuit.Circuit {
	return i.topology
}

//GetGroups returns the group used by the server
func (i *Instance) GetGroup() *cyclic.Group {
	return i.params.Groups.CMix
}

//GetUserRegistry returns the user registry used by the server
func (i *Instance) GetUserRegistry() globals.UserRegistry {
	return i.userReg
}

//GetRoundManager returns the round manager
func (i *Instance) GetRoundManager() *round.Manager {
	return i.roundManager
}

//GetResourceQueue returns the resource queue used by the serverequals
func (i *Instance) GetResourceQueue() *ResourceQueue {
	return i.resourceQueue
}

func (i *Instance) GetNetwork() *node.NodeComms {
	return i.network
}

//GetID returns this node's ID
func (i *Instance) GetID() *id.Node {
	return i.thisNode
}

//GetPubKey returns the server DSA public key
func (i *Instance) GetPubKey() *signature.DSAPublicKey {
	return i.pubKey
}

//GetPrivKey returns the server DSA private key
func (i *Instance) GetPrivKey() *signature.DSAPrivateKey {
	return i.privKey
}

//GetSkipReg returns the skipReg parameter
func (i *Instance) GetSkipReg() bool {
	return i.params.SkipReg
}

//GetRegServerPubKey returns the public key of the registration server
func (i *Instance) GetRegServerPubKey() *signature.DSAPublicKey {
	return i.regServerPubKey
}

//GetBatchSize returns the batch size
func (i *Instance) GetBatchSize() uint32 {
	return i.params.Batch
}

// FIXME Populate this from the YAML or something
func (i *Instance) GetGraphGenerator() services.GraphGenerator {
	return i.graphGenerator
}

//Initializes the first node components of the instance
func (i *Instance) InitFirstNode() {
	i.firstNode.Initialize()
}

//Initializes the last node components of the instance
func (i *Instance) InitLastNode() {
	i.LastNode.Initialize()
}

//IsFirstNode returns if the node is first node
func (i *Instance) IsFirstNode() bool {
	return i.topology.IsFirstNode(i.thisNode)
}

//IsLastNode returns if the node is last node
func (i *Instance) IsLastNode() bool {
	return i.topology.IsLastNode(i.thisNode)
}

// Create a server instance. To actually kick off the server,
// call RunFirstNode() on the resulting ServerInstance.
func CreateServerInstance(params *conf.Params, db globals.UserRegistry,
	publicKey *signature.DSAPublicKey, privateKey *signature.DSAPrivateKey) *Instance {

	//TODO: build system wide error handeling
	PanicHandler := func(g, m string, err error) {
		panic(fmt.Sprintf("Error in module %s of graph %s: %s", g, m, err.Error()))
	}

	instance := Instance{
		roundManager:  round.NewManager(),
		params:        params,
		resourceQueue: initQueue(),
		userReg:       db,
		//FIXME: make this smarter
		graphGenerator: services.NewGraphGenerator(4, PanicHandler,
			uint8(runtime.NumCPU()), 4, 0.0),
	}

	// Create the topology that will be used for all rounds
	// Each nodeID should be base64 encoded in the yaml
	var nodeIDs []*id.Node
	var nodeIDDecodeErrorHappened bool
	for i := range params.NodeIDs {
		nodeID, err := base64.StdEncoding.DecodeString(params.NodeIDs[i])
		if err != nil {
			// This indicates a server misconfiguration which needs fixing for
			// the server to function properly
			err = errors.Wrapf(err, "Node ID at index %v failed to decode", i)
			jww.ERROR.Print(err)
			nodeIDDecodeErrorHappened = true
		}
		nodeIDs = append(nodeIDs, id.NewNodeFromBytes(nodeID))
	}
	if nodeIDDecodeErrorHappened {
		jww.ERROR.Panic("One or more node IDs didn't base64 decode correctly")
	}

	if params.RegServerPK != "" {
		grp := instance.params.Groups.CMix
		dsaParams := signature.CustomDSAParams(grp.GetP(), grp.GetQ(), grp.GetG())

		block, _ := pem.Decode([]byte(params.RegServerPK))

		if block == nil || block.Type != "PUBLIC KEY" {
			jww.ERROR.Panic("Registration Server Public Key did not " +
				"decode correctly")
		}

		instance.regServerPubKey = signature.ReconstructPublicKey(dsaParams,
			large.NewIntFromBytes(block.Bytes))
	} else {
		jww.WARN.Print("No registration key given, registration not possible")
	}

	instance.topology = circuit.New(nodeIDs)
	instance.thisNode = instance.topology.GetNodeAtIndex(params.Index)

	// Create a node id object with the random bytes
	// Generate DSA Private/Public key pair
	instance.pubKey = publicKey
	instance.privKey = privateKey
	// Hardcoded registration server publicKey
	// TODO: For now set this to false, but value should come from config file
	instance.params.SkipReg = false

	return &instance
}

// GenerateId generates a random ID and returns it
// FIXME: This function needs to be replaced
func GenerateId() *id.Node {

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

// Initializes the network on this server instance
// After the network object is created, you still need to use it to connect
// to other servers in the network using ConnectToNode or ConnectToGateway.
// Additionally, to clean up the network object (especially in tests), call
// Shutdown() on the network object.
func (i *Instance) InitNetwork(
	makeImplementation func(*Instance) *node.Implementation) *node.NodeComms {
	addr := i.params.NodeAddresses[i.params.Index]
	i.network = node.StartNode(addr, makeImplementation(i), i.params.Path.Cert,
		i.params.Path.Key)

	var tlsCert credentials.TransportCredentials

	if i.params.Path.Cert != "" {
		tlsCert = connect.NewCredentialsFromFile(i.params.Path.Cert, "")
	} else {
		jww.WARN.Printf("Starting node without TLS credentials")
	}

	for x := 0; x < len(i.params.NodeIDs); x++ {
		i.network.ConnectToNode(i.topology.GetNodeAtIndex(x), i.params.NodeAddresses[x],
			tlsCert)
	}

	if i.params.Gateways != nil {
		i.network.ConnectToGateway(i.thisNode.NewGateway(),
			i.params.Gateways[i.params.Index], tlsCert)
	} else {
		jww.WARN.Printf("No Gateway avalible, starting without gateway")
	}

	jww.INFO.Printf("Network Interface Initilized for Node ")

	return i.network
}

func (i *Instance) Run() {
	go i.resourceQueue.run(i)
}

func (i *Instance) String() string {
	id := i.thisNode
	numNodes := i.topology.Len()
	myLoc := i.topology.GetNodeLocation(id)
	// TODO: IP Address an dlistening port would be helpful!
	ipAddr := "HostUnknown:PortUnknown"
	return fmt.Sprintf("%s - (%d/%d)", ipAddr, myLoc, numNodes)
}
