package round

import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server/phase"
	"sync/atomic"
)

var ErrRoundDoestNotHaveResponse = errors.New("The round does not a response to the given input")
var ErrPhaseInIncorrectStateToContinue = errors.New("The phase in the given round is not " +
	"at the correct state to proceed")

type Round struct {
	id     id.Round
	buffer *Buffer

	topology *circuit.Circuit
	state    *uint32

	//on first node and last node the phases vary
	phaseMap       map[phase.Type]int
	phases         []phase.Phase
	numPhaseStates uint32

	//holds responses to coms, how to check and process incoming comms
	responses phase.ResponseMap
}

// Creates and initializes a new round, including all phases, topology,
// and batchsize
func New(grp *cyclic.Group, id id.Round, phases []phase.Phase, responses phase.ResponseMap,
	circut *circuit.Circuit, nodeID *id.Node, batchSize uint32) *Round {

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
			if currentState < 0 {
				return 0
			} else if currentState > int64(phase.NumStates)-2 {
				return phase.NumStates - 1
			} else {
				return phase.State(currentState + 1)
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

	round.phases = make([]phase.Phase, len(phases))

	copy(round.phases[:], phases[:])

	round.topology = circut

	if round.topology.IsLastNode(nodeID) {
		round.buffer.InitLastNode()
	}

	round.responses = responses

	//set the state of the first phase to available
	success := atomic.CompareAndSwapUint32(round.state, uint32(phase.Initialized), uint32(phase.Available))
	if !success {
		jww.FATAL.Println("CMixPhase state initialization failed")
	}

	return &round
}

func (r *Round) GetID() id.Round {
	return r.id
}

func (r *Round) GetBuffer() *Buffer {
	return r.buffer
}

func (r *Round) GetPhase(p phase.Type) (phase.Phase, error) {
	i, ok := r.phaseMap[p]
	if !ok || i >= len(r.phases) || r.phases[i] == nil {
		return nil, errors.Errorf("Round %s missing phase type %s",
			r, p)
	}
	return r.phases[i], nil
}

func (r *Round) GetCurrentPhase() phase.Phase {
	phase := atomic.LoadUint32(r.state) / uint32(phase.NumStates)
	return r.phases[phase]
}

func (r *Round) GetTopology() *circuit.Circuit {
	return r.topology
}

func (r *Round) HandleIncomingComm(commTag string) (phase.Phase, error) {
	response, ok := r.responses[commTag]

	if !ok {
		return nil, errors.WithMessage(ErrRoundDoestNotHaveResponse,
			fmt.Sprintf("Round: %v, Input: %s", r.id, commTag))
	}

	phaseToCheck, err := r.GetPhase(response.GetPhaseLookup())

	if err != nil {
		jww.FATAL.Panicf("CMixPhase %s looked up up from response map "+
			"does not exist in round", response.GetPhaseLookup())
	}

	if response.CheckState(phaseToCheck.GetState()) {
		returnPhase, err := r.GetPhase(response.GetReturnPhase())
		if err != nil {
			jww.FATAL.Panicf("The requested phase could not be returned in the comm handler")
		}

		return returnPhase, nil
	} else {
		return nil, ErrPhaseInIncorrectStateToContinue
	}
}

// String stringer interface implementation for rounds.
// TODO: Maybe print active conns for this round or other data?
func (r *Round) String() string {
	currentPhase := r.GetCurrentPhase()
	return fmt.Sprintf("%d (%d - %s)", r.id, r.state, currentPhase)
}
