package server

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/server/node"
	"sync"
	"sync/atomic"
)

type NodeAddress struct {
	Cert    string
	Address string
	Id      uint64
}

func (na NodeAddress) DeepCopy() NodeAddress {
	return NodeAddress{na.Cert, na.Address, na.Id}
}

type Round struct {
	id     node.RoundID
	buffer *node.RoundBuffer

	nodes []NodeAddress
	myLoc int

	//on first node and last node the phases vary
	phaseMap     map[node.PhaseType]int
	phases       []*Phase
	phaseStates  []*PhaseState
	currentPhase *node.PhaseType
	phaseStateRW sync.RWMutex
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
	if int(p) > len(r.phases) {
		return nil
	}
	return r.phases[r.phaseMap[p]]
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

func (r *Round) FinishPhase(phase node.PhaseType) {
	p := r.phaseMap[phase]
	r.phaseStateRW.Lock()
	success := atomic.CompareAndSwapUint32((*uint32)(r.currentPhase), (uint32)(p), (uint32)(p)+1)
	if !success {
		jww.FATAL.Panicf("Phase incremented incorrectly from %v as if %v in round %v",
			atomic.LoadUint32((*uint32)(r.currentPhase)), p, r.id)
	}

	success = atomic.CompareAndSwapUint32((*uint32)(r.phaseStates[p]), uint32(Running), uint32(Completed))
	if !success {
		jww.FATAL.Panicf("Phase state of running phase %s could not be incremented to Completed", phase.String())
	}

	if p+1 < len(r.phases) {
		success = atomic.CompareAndSwapUint32((*uint32)(r.phaseStates[p]), uint32(Initialized), uint32(Available))
		if !success {
			jww.FATAL.Panicf("Phase state of new phase %s could not be incremented to Avalable", (phase + 1).String())
		}
	}
	r.phaseStateRW.Unlock()
}
