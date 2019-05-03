package phase

import (
	"fmt"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/services"
	"sync"
	"time"
)

// Holds a single phase to be executed by the server in a round
type Phase struct {
	graph               *services.Graph
	tYpe                Type
	transmissionHandler Transmit
	timeout             time.Duration

	roundID           id.Round
	roundIDset        sync.Once
	transitionToState Transition
	getState          GetState
}

// New makes a new phase with the given graph, phase.Name, transmission handler, and timeout
func New(g *services.Graph, name Type, tHandler Transmit, timeout time.Duration) *Phase {
	return &Phase{
		graph:               g,
		tYpe:                name,
		transmissionHandler: tHandler,
		timeout:             timeout,
	}
}

/*Setters */
// SetRoundIDOnce sets the round ID.  Can only be called once.
// Should only be called from Round package that initializes states
// Must be called on all phases in their order in the round
func (p *Phase) ConnectToRound(id id.Round, setState Transition,
	getState GetState) {
	p.roundIDset.Do(func() {
		p.roundID = id
		p.transitionToState = setState
		p.getState = getState
	})
}

/*Getters*/
// GetGraph gets the graph associated with the phase
func (p *Phase) GetGraph() *services.Graph {
	return p.graph
}

func (p *Phase) GetRoundID() id.Round {
	return p.roundID
}

func (p *Phase) GetType() Type {
	return p.tYpe
}

// GetState returns the current state of the phase
func (p *Phase) GetState() State {
	return p.getState()
}

func (p *Phase) TransitionTo(newState State) bool {
	return p.transitionToState(newState)
}

// GetTransmissionHandler returns the phase's transmission handling function
func (p *Phase) GetTransmissionHandler() Transmit {
	return p.transmissionHandler
}

// GetTimeout gets the timeout at which the phase will fail
func (p *Phase) GetTimeout() time.Duration {
	return p.timeout
}

/*Utility*/
// Cmp checks if two phases are the same
func (p *Phase) Cmp(p2 *Phase) bool {
	return p.roundID == p2.roundID && p.tYpe == p2.tYpe
}

//String adheres to the string interface
func (p *Phase) String() string {
	return fmt.Sprintf("phase.Phase{roundID: %v, phaseType: %s}",
		p.roundID, p.tYpe)
}

// ReadyToReceiveData returns true if the phase can receive data
func (p *Phase) ReadyToReceiveData() bool {
	phaseState := p.GetState()
	return phaseState == Available || phaseState == Queued || phaseState == Running
}
