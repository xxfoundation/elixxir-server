package server

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/node"
	"sync"
	"sync/atomic"
)

type NodeAddress struct {
	cert    string
	address string
	id      uint64
}

func (na NodeAddress) DeepCopy() NodeAddress {
	return NodeAddress{na.cert, na.address, na.id}
}

type PhaseState uint32

const (
	//Initialized: Data structures for the phase have been created but it is not ready to run
	Initialized PhaseState = iota
	//Available: Next phase to run according to round but no input has been received
	Available
	//Queued: Next phase to run according to round and input has been received but it
	// has not begun execution by resource manager
	Queued
	//Running: Next phase to run according to round and input has been received and it
	// is being executed by resource manager
	Running
	//Complete: Phase is complete
	Completed
)

type Round struct {
	id     node.RoundID
	buffer *node.RoundBuffer

	nodes []NodeAddress
	myLoc int

	phases       [node.NUM_PHASES]*Phase
	phaseStates  [node.NUM_PHASES]*PhaseState
	currentPhase *node.PhaseType
	phaseStateRW sync.RWMutex
}

func newRound(grp *cyclic.Group, id node.RoundID, phases [node.NUM_PHASES]*Phase,
	nodes []NodeAddress, myLoc int, batchsize uint32) *Round {

	round := Round{}

	maxBatchSize := uint32(0)

	for _, p := range phases {
		p.Round = &round
		if p.Graph.GetExpandedBatchSize() > maxBatchSize {
			maxBatchSize = p.Graph.GetExpandedBatchSize()
		}
	}

	round.id = id
	round.buffer = node.NewRound(grp, batchsize, maxBatchSize)

	round.phaseStateRW.Lock()
	defer round.phaseStateRW.Unlock()

	for index, p := range phases {
		p.Graph.Link(&round)
		phaseState := Initialized
		round.phaseStates[index] = &phaseState
	}

	copy(round.phases[:], phases[:])

	round.nodes = make([]NodeAddress, len(nodes))
	for itr := range round.nodes {
		round.nodes[itr] = nodes[itr].DeepCopy()
	}

	round.myLoc = myLoc

	phase := node.PhaseType(0)
	round.currentPhase = &phase

	success := atomic.CompareAndSwapUint32((*uint32)(round.phaseStates[0]),
		uint32(Initialized), uint32(Available))

	if !success {
		jww.FATAL.Panicf("Could not set the state on a newly initilized phase in new round")
	}

	return &round
}

func (r *Round) GetNextNodeAddress() NodeAddress {
	return r.nodes[(r.myLoc+1)%len(r.nodes)]
}

func (r *Round) GetPrevNodeAddress() NodeAddress {
	return r.nodes[(r.myLoc-1)%len(r.nodes)]
}

func (r *Round) GetNodeAddress(index int) NodeAddress {
	return r.nodes[index%len(r.nodes)]
}

func (r *Round) GetAllNodesAddress() []NodeAddress {
	nal := make([]NodeAddress, len(r.nodes))

	for i := range nal {
		nal[i] = r.nodes[i].DeepCopy()
	}
	return nal
}

func (r *Round) GetID() node.RoundID {
	return r.id
}

func (r *Round) GetBuffer() *node.RoundBuffer {
	return r.buffer
}

func (r *Round) GetPhase(p node.PhaseType) *Phase {
	if p > node.NUM_PHASES {
		return nil
	}
	return r.phases[p]
}

func (r *Round) GetPhaseState(p node.PhaseType) PhaseState {
	r.phaseStateRW.RLock()
	state := PhaseState(atomic.LoadUint32((*uint32)(r.phaseStates[p])))
	r.phaseStateRW.RUnlock()
	return state
}

func (r *Round) IncrementPhaseToQueued(p node.PhaseType) bool {
	r.phaseStateRW.RLock()
	success := atomic.CompareAndSwapUint32((*uint32)(r.phaseStates[p]), uint32(Available), uint32(Queued))
	r.phaseStateRW.RUnlock()
	return success
}

func (r *Round) IncrementPhaseToRunning(p node.PhaseType) bool {
	r.phaseStateRW.RLock()
	success := atomic.CompareAndSwapUint32((*uint32)(r.phaseStates[p]), uint32(Queued), uint32(Running))
	r.phaseStateRW.RUnlock()
	return success
}

func (r *Round) FinishPhase(p node.PhaseType) {
	r.phaseStateRW.Lock()
	success := atomic.CompareAndSwapUint32((*uint32)(r.currentPhase), (uint32)(p), (uint32)(p)+1)
	if !success {
		jww.FATAL.Panicf("Phase incremented incorrectly from %v as if %v in round %v",
			atomic.LoadUint32((*uint32)(r.currentPhase)), p, r.id)
	}

	success = atomic.CompareAndSwapUint32((*uint32)(r.phaseStates[p]), uint32(Running), uint32(Completed))
	if !success {
		jww.FATAL.Panicf("Phase state of running phase %s could not be incremented to Completed", p.String())
	}

	if p+1 < node.NUM_PHASES {
		success = atomic.CompareAndSwapUint32((*uint32)(r.phaseStates[p]), uint32(Initialized), uint32(Available))
		if !success {
			jww.FATAL.Panicf("Phase state of new phase %s could not be incremented to Avalable", (p + 1).String())
		}
	}
	r.phaseStateRW.Unlock()
}
