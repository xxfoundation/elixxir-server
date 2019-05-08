package server

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/server/round"
)

// Holds long-lived server state
type Instance struct {
	id            *id.Node
	roundManager  *round.Manager
	network       *node.NodeComms
	resourceQueue *ResourceQueue
	grp           *cyclic.Group
	userReg       globals.UserRegistry
	firstNode
	lastNode
}

//GetGroup returns the group used by the server
func (i *Instance) GetGroup() *cyclic.Group {
	return i.grp
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

//Initializes the first node components of the instance
func (i *Instance) InitFirstNode() {
	i.firstNode.Initialize()
}

//Initializes the last node components of the instance
func (i *Instance) InitLastNode() {
	i.lastNode.Initialize()
}

// Create a server instance. To actually kick off the server,
// call Run() on the resulting ServerIsntance.
func CreateServerInstance(grp *cyclic.Group, db globals.UserRegistry) *Instance {
	instance := Instance{
		roundManager: round.NewManager(),
		grp:          grp,
	}
	instance.resourceQueue = initQueue()
	instance.userReg = db

	//Generate a random node id as a placeholder
	nodeIdBytes := make([]byte, id.NodeIdLen)
	rng := csprng.NewSystemRNG()
	_, err := rng.Read(nodeIdBytes)
	if err != nil {
		err := errors.New(err.Error())
		jww.FATAL.Panicf("Could not generate random nodeID: %+v", err)
	}

	nid := &id.Node{}
	nid.SetBytes(nodeIdBytes)
	instance.id = nid

	return &instance
}

// TODO(sb) Should there be a version of this that uses the network definition
//  file to create all the connections in the network?
// Initializes the network on this server instance
// After the network object is created, you still need to use it to connect
// to other servers in the network using ConnectToNode or ConnectToGateway.
// Additionally, to clean up the network object (especially in tests), call
// Shutdown() on the network object.
func (i *Instance) InitNetwork(addr string,
	makeImplementation func(*Instance) *node.Implementation,
	certPath string, keyPath string) *node.NodeComms {
	i.network = node.StartNode(addr, makeImplementation(i), certPath, keyPath)
	return i.network
}

func (i *Instance) Run() {
	go queueRunner(i)
}

//GetID returns the nodeID
func (i *Instance) GetID() *id.Node {
	return i.id.DeepCopy()
}
