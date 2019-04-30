package round

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/services"
	"sync/atomic"
)

type Round struct {
	id     id.Round
	buffer *Buffer

	nodeAddressList *services.NodeAddressList
	state           *uint32

	//on first node and last node the phases vary
	phaseMap map[phase.Type]int
	phases   []*phase.Phase
}

// Creates and initializes a new round, including all phases
func New(grp *cyclic.Group, id id.Round, phases []*phase.Phase, nodes []services.NodeAddress, myLoc int, batchSize uint32) *Round {

	round := Round{}
	round.id = id

	maxBatchSize := uint32(0)

	state := uint32(0)
	round.state = &state

	for index, p := range phases {
		p.GetGraph().Build(batchSize)
		if p.GetGraph().GetExpandedBatchSize() > maxBatchSize {
			maxBatchSize = p.GetGraph().GetExpandedBatchSize()
		}

		localStateOffset := uint32(index) * uint32(phase.NumStates)

		//build the function this phase will use to increment it's state
		increment := func(to phase.State) bool {
			newState := localStateOffset + uint32(to)
			expectedOld := newState - 1
			return atomic.CompareAndSwapUint32(round.state, expectedOld, newState)
		}

		//build the function this phase will use to get its state
		get := func() phase.State {
			curentState := int64(atomic.LoadUint32(round.state)) - int64(localStateOffset)
			if curentState <= int64(phase.Initialized) {
				return phase.Initialized
			} else if curentState >= int64(phase.Finished) {
				return phase.Finished
			} else {
				return phase.State(curentState)
			}
		}

		//connect the phase to the round passing its state accessor functions
		p.ConnectToRound(id, increment, get)
	}

	round.buffer = NewBuffer(grp, batchSize, maxBatchSize)
	round.phaseMap = make(map[phase.Type]int)

	// this phasemap logic looks suspicious
	for index, p := range phases {
		p.GetGraph().Link(grp, &round)
		round.phaseMap[p.GetType()] = index
	}

	round.phases = make([]*phase.Phase, len(phases))

	copy(round.phases[:], phases[:])

	round.nodeAddressList = services.NewNodeAddressList(nodes, myLoc)

	//set the state of the first phase to available
	success := atomic.CompareAndSwapUint32(round.state, uint32(phase.Initialized), uint32(phase.Available))
	if !success {
		jww.FATAL.Println("Phase state initialization failed")
	}

	return &round
}

func (r *Round) GetID() id.Round {
	return r.id
}

func (r *Round) GetBuffer() *Buffer {
	return r.buffer
}

func (r *Round) GetPhase(p phase.Type) *phase.Phase {
	i, ok := r.phaseMap[p]
	if !ok {
		return nil
	} else {
		return r.phases[i]
	}
}

func (r *Round) GetCurrentPhase() *phase.Phase {
	phase := atomic.LoadUint32(r.state) / uint32(phase.NumStates)
	return r.phases[phase]
}

func (r *Round) GetNodeAddressList() *services.NodeAddressList {
	return r.nodeAddressList
}
