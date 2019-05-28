package server

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/server/conf"
	"gitlab.com/elixxir/server/server/round"
)

// Holds long-lived server state
type Instance struct {
	id            *id.Node
	roundManager  *round.Manager
	network       *node.NodeComms
	resourceQueue *ResourceQueue
	userReg       globals.UserRegistry
	pubKey        *signature.DSAPublicKey
	privKey       *signature.DSAPrivateKey
	regPubKey     *signature.DSAPublicKey
	firstNode
	lastNode
	params conf.Params
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

//GetID returns the nodeID
func (i *Instance) GetID() *id.Node {
	return i.id.DeepCopy()
}

//GetPubKey returns the server DSA public key
func (i *Instance) GetPubKey() *signature.DSAPublicKey {
	return i.pubKey
}

//GetPrivKey returns the server DSA private key
func (i *Instance) GetPrivKey() *signature.DSAPrivateKey {
	return i.privKey
}

//GetRegPubKey returns the registration server DSA public key
func (i *Instance) GetRegPubKey() *signature.DSAPublicKey {
	return i.regPubKey
}

//GetSkipReg returns the skipReg parameter
func (i *Instance) GetSkipReg() bool {
	return i.params.SkipReg
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
func CreateServerInstance(params conf.Params, nid *id.Node, db globals.UserRegistry) *Instance {
	instance := Instance{
		roundManager:  round.NewManager(),
		params:        params,
		id:            nid,
		resourceQueue: initQueue(),
		userReg:       db,
	}

	// Create a node id object with the random bytes
	// Generate DSA Private/Public key pair
	rng := csprng.NewSystemRNG()
	grp := params.Groups.CMix
	dsaParams := signature.CustomDSAParams(grp.GetP(), grp.GetQ(), grp.GetG())
	instance.privKey = dsaParams.PrivateKeyGen(rng)
	instance.pubKey = instance.privKey.PublicKeyGen()
	// Hardcoded registration server publicKey
	instance.regPubKey = signature.ReconstructPublicKey(dsaParams,
		large.NewIntFromString("1ae3fd4e9829fb464459e05bca392bec5067152fb43a569ad3c3b68bbcad84f0"+
			"ff8d31c767da3eabcfc0870d82b39568610b52f2b72b493bbede6e952c9a7fd4"+
			"4a8161e62a9046828c4a65f401b2f054ebf7376e89dab547d8a3c3d46891e78a"+
			"cfc4015713cbfb5b0b6cab0f8dfb46b891f3542046ace4cab984d5dfef4f52d4"+
			"347dc7e52f6a7ea851dda076f0ed1fef86ec6b5c2a4807149906bf8e0bf70b30"+
			"1147fea88fd95009edfbe0de8ffc1a864e4b3a24265b61a1c47a4e9307e7c84f"+
			"9b5591765b530f5859fa97b22ce9b51385d3d13088795b2f9fd0cb59357fe938"+
			"346117df2acf2bab22d942de1a70e8d5d62fc0e99d8742a0f16df94ce3a0abbb", 16))
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
	go i.resourceQueue.run(i)
}
