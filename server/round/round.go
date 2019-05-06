package round

import (
	"fmt"
	"github.com/pkg/errors"
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

		// the NumStates-2 exists because the Initialized and verified states
		// are done implicitly as less then available / greater then computed
		// the +1 exists because the system needs an initialized state for the
		// first phase
		localStateOffset := uint32(index)*uint32(phase.NumStates-2) + 1

		// Build the function this phase will use to increment its state
		increment := func(from, to phase.State) bool {
			if from >= to {
				jww.FATAL.Panicf("Cannot incremeent backwards from %s to %s",
					from, to)
			}
			// 1 is subtracted because Initialized doesnt hold a true state
			newState := localStateOffset + uint32(to) - 1
			expectedOld := localStateOffset + uint32(from) - 1
			return atomic.CompareAndSwapUint32(round.state, expectedOld, newState)
		}

		// Build the function this phase will use to get its state
		// -1 is at the end of all phase states because Initialized
		// is not counted as a state
		get := func() phase.State {
			currentState := int64(atomic.LoadUint32(round.state)) - int64(localStateOffset)
			if currentState < int64(phase.Available)-1 {
				return phase.Initialized
			} else if currentState > int64(phase.Computed)-1 {
				return phase.Verified
			} else {
				return phase.State(currentState) + 1
			}
		}

		// Connect the phase to the round passing its state accessor functions
		p.ConnectToRound(id, increment, get)
	}

	round.buffer = NewBuffer(grp, batchSize, maxBatchSize)
	round.phaseMap = make(map[phase.Type]int)

	for index, p := range phases {
		p.GetGraph().Link(grp, &round)
		round.phaseMap[p.GetType()] = index
	}

	round.phases = make([]*phase.Phase, len(phases))

	copy(round.phases[:], phases[:])

	round.nodeAddressList = services.NewNodeAddressList(nodes, myLoc)

	if round.nodeAddressList.IsLastNode() {
		round.buffer.InitLastNode()
	}

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

func (r *Round) GetPhase(p phase.Type) (*phase.Phase, error) {
	i, ok := r.phaseMap[p]
	if !ok || i >= len(r.phases) || r.phases[i] == nil {
		return nil, errors.Errorf("Round %s missing phase type %s",
			r, p)
	}
	return r.phases[i], nil
}

func (r *Round) GetCurrentPhase() *phase.Phase {
	phase := atomic.LoadUint32(r.state) / uint32(phase.NumStates)
	return r.phases[phase]
}

func (r *Round) GetNodeAddressList() *services.NodeAddressList {
	return r.nodeAddressList
}

// String stringer interface implementation for rounds.
// TODO: Maybe print active conns for this round or other data?
func (r *Round) String() string {
	currentPhase := r.GetCurrentPhase()
	return fmt.Sprintf("%d (%d - %s)", r.id, r.state, currentPhase)
}
