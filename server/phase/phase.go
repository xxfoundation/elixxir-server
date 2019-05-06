package phase

import (
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/services"
	"sync/atomic"
	"time"
)

// Holds a single phase to be executed by the server in a round
type Phase struct {
	graph               *services.Graph
	tYpe                Type
	transmissionHandler Transmit
	timeout             time.Duration

	roundID           id.Round
	connected         *uint32
	transitionToState Transition
	getState          GetState

	//This bool denotes if the phase goes straight to completed or waits for an
	//External check at Computed
	verification bool
}

// New makes a new phase with the given graph, phase.Name, transmission handler, and timeout
func New(g *services.Graph, name Type, tHandler Transmit, timeout time.Duration) *Phase {
	connected := uint32(0)
	return &Phase{
		graph:               g,
		tYpe:                name,
		transmissionHandler: tHandler,
		timeout:             timeout,
		connected:           &connected,
	}
}

/* Setters */
// EnableVerification sets the internal variable phase.verification to true which
// ensures the system will require an extra state before completing the phase
func (p *Phase) EnableVerification() {
	if atomic.LoadUint32(p.connected) == 0 {
		p.verification = true
	} else {
		jww.FATAL.Printf("Cannot set verification to true on phase %s"+
			"Because it is connected to round %v",
			p.GetType(), p.GetRoundID())
	}
}

// ConnectToRound sets the round ID.  Can only be called once.
// Should only be called from Round package that initializes states
// Must be called on all phases in their order in the round
func (p *Phase) ConnectToRound(id id.Round, setState Transition,
	getState GetState) {
	numSet := atomic.AddUint32(p.connected, 1)
	if numSet == 1 {
		p.roundID = id
		p.transitionToState = setState
		p.getState = getState
	} else {
		jww.FATAL.Printf("Cannot connect phase %s to round %v: numset=%v",
			p.GetType(), p.GetRoundID(), numSet)
	}
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

func (p *Phase) TransitionToAvailable() bool {
	return p.transitionToState(Available)
}

func (p *Phase) TransitionToQueued() bool {
	return p.transitionToState(Queued)
}

func (p *Phase) TransitionToRunning() bool {
	return p.transitionToState(Running)
}

func (p *Phase) Finish() bool {
	success := p.transitionToState(Computed)

	if !success {
		jww.FATAL.Panicf("Phase %s of round %v at incorrect state"+
			"to be transitioned to Computed", p.tYpe, p.roundID)
	}

	if !p.verification {
		success = p.transitionToState(Verified)
		if !success {
			jww.FATAL.Panicf("Phase %s of round %v at incorrect state"+
				"to be transitioned to Verified", p.tYpe, p.roundID)
		}
		return true
	}

	return false
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

// ReadyToVerify returns true if the phase is in computed state
// and is ready to be verified
func (p *Phase) ReadyToVerify() bool {
	phaseState := p.GetState()
	return phaseState == Computed
}
