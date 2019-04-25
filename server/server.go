package server

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/node"
	"sync"
	"sync/atomic"
)

// Holds long-lived server state
type Instance struct {
	roundManager  *RoundManager
	resourceQueue *ResourceQueue
	grp           *cyclic.Group
	userReg       globals.UserRegistry
}

func (i *Instance) GetGroup() *cyclic.Group {
	return i.grp
}

func (i *Instance) GetUserRegistry() globals.UserRegistry {
	return i.userReg
}

func (i *Instance) GetRoundManager() *RoundManager {
	return i.roundManager
}

func (i *Instance) GetResourceQueue() *ResourceQueue {
	return i.resourceQueue
}

// Create a server instance. To actually kick off the server,
// call Run() on the resulting ServerIsntance.
func CreateServerInstance(grp *cyclic.Group, db globals.UserRegistry) *Instance {
	instance := Instance{
		roundManager: &RoundManager{
			roundMap: &sync.Map{},
		},
		grp: grp,
	}
	instance.resourceQueue = &ResourceQueue{
		// these are the phases
		phaseQueue: make(chan *Phase, 5000),
		// there will only active phase, and this channel is used to kill it
		finishChan: make(chan PhaseFingerprint, 1),
	}
	instance.userReg = db
	return &instance
}

func (i *Instance) Run() {
	go queueRunner(i.resourceQueue)
}

// Creates and initializes a new round, including all phases
// Also, adds the round to the round manager
func (i *Instance) CreateRound(id node.RoundID,
	phases []*Phase, nodes []NodeAddress, myLoc int, batchSize uint32) {

	round := Round{}

	maxBatchSize := uint32(0)

	for _, p := range phases {
		p.Round = &round
		if p.Graph.GetExpandedBatchSize() > maxBatchSize {
			maxBatchSize = p.Graph.GetExpandedBatchSize()
		}
	}

	round.id = id
	round.buffer = node.NewRound(i.grp, batchSize, maxBatchSize)

	round.phaseStateRW.Lock()
	defer round.phaseStateRW.Unlock()

	for index, p := range phases {
		p.Graph.Link(&round)
		phaseState := Initialized
		round.phaseStates[index] = &phaseState
		round.phaseMap[p.Phase] = index
	}

	copy(round.phases[:], phases[:])

	round.nodes = make([]NodeAddress, len(nodes))
	for idx := range round.nodes {
		round.nodes[idx] = nodes[idx].DeepCopy()
	}

	round.myLoc = myLoc

	phase := node.PhaseType(0)
	round.currentPhase = &phase

	success := atomic.CompareAndSwapUint32((*uint32)(round.phaseStates[0]),
		uint32(Initialized), uint32(Available))

	if !success {
		jww.FATAL.Panic("Could not set the state on a newly initialized" +
			" phase in new round")
	}

	i.roundManager.AddRound(&round)
}
