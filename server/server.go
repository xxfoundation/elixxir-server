package server

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/server/round"
)

// Holds long-lived server state
type Instance struct {
	id            *id.Node
	roundManager  *round.Manager
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
	instance.resourceQueue = &ResourceQueue{
		// these are the phases
		phaseQueue: make(chan *phase.Phase, 5000),
		// there will only active phase, and this channel is used to kill it
		finishChan: make(chan *phase.Phase, 1),
	}
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

func (i *Instance) Run() {
	go queueRunner(i)
}

//GetID returns the nodeID
func (i *Instance) GetID() *id.Node {
	return i.id.DeepCopy()
}

func (i *Instance) HandleIncomingPhase(roundID id.Round, phaseType phase.Type) (*phase.Phase, error) {
	// Get the phase (with error checking) from the round manager by looking
	// up the round
	p, err := i.roundManager.GetPhase(roundID, int32(phaseType))
	if err != nil {
		return nil, err
	}

	// If the phase can't receive data this is a fatal error, not a
	// blocking issue.
	if !p.ReadyToReceiveData() {
		return nil, errors.Errorf("Phase %s, round %d is not ready!",
			p, roundID)
	}

	// Update queue to tell it we are running this phase
	i.resourceQueue.UpsertPhase(p)

	return p, nil
}
