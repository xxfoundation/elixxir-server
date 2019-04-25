package server

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/node"
	"sync"
	"sync/atomic"
)

type ServerInstance struct {
	roundManager  *RoundManager
	resourceQueue *ResourceQueue
	grp           *cyclic.Group
	userDB        *globals.UserDB
}

func CreateServerInstance(db *globals.UserDB) *ServerInstance {
	instance := ServerInstance{
		roundManager: &RoundManager{
			roundMap: &sync.Map{},
		},
		grp: &cyclic.Group{},
	}
	instance.resourceQueue = &ResourceQueue{
		// these are the phases
		phaseQueue: make(chan *Phase, 5000),
		// there will only active phase, and this channel is used to kill it
		finishChan: make(chan PhaseFingerprint, 1),
	}
	instance.userDB = db
	go queueRunner(instance.resourceQueue)
	return &instance
}

// Creates and initializes a new round, including all phases
func (s *ServerInstance) CreateRound(id node.RoundID,
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
	round.buffer = node.NewRound(s.grp, batchSize, maxBatchSize)

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

	s.roundManager.AddRound(&round)
}
