package phase

import (
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/services"
	"sync"
	"sync/atomic"
	"time"
)

// Holds a single phase to be executed by the server in a round
type Phase struct {
	graph               *services.Graph
	roundID             id.Round
	roundIDset          sync.Once
	name                Name
	state               *uint32
	transmissionHandler server.Transmission
	timeout             time.Duration
}

// New makes a new phase with the given graph, phase.Name, transmission handler, and timeout
func New(g *services.Graph, name Name, tHandler server.Transmission, timeout time.Duration) *Phase {
	state := uint32(Initialized)
	return &Phase{
		graph:               g,
		name:                name,
		transmissionHandler: tHandler,
		timeout:             timeout,
		state:               &state,
	}
}

/*Setters */
// SetRoundIDOnce sets the round ID.  Can only be called once.
func (p *Phase) SetRoundIDOnce(id id.Round) {
	p.roundIDset.Do(func() { p.roundID = id })
}

/*Getters*/
// GetGraph gets the graph associated with the phase
func (p *Phase) GetGraph() *services.Graph {
	return p.graph
}

func (p *Phase) GetRoundID() id.Round {
	return p.roundID
}

func (p *Phase) GetName() Name {
	return p.name
}

// GetState returns the current state of the phase
func (p *Phase) GetState() State {
	return State(atomic.LoadUint32(p.state))
}

// GetTransmissionHandler returns the phase's transmission handling function
func (p *Phase) GetTransmissionHandler() server.Transmission {
	return p.transmissionHandler
}

// GetTimeout gets the timeout at which the phase will fail
func (p *Phase) GetTimeout() time.Duration {
	return p.timeout
}

/*Utility*/
// ReadyToReceiveData returns true if the phase can receive data
func (p *Phase) ReadyToReceiveData() bool {
	phaseState := p.GetState()
	return phaseState == Available || phaseState == Queued || phaseState == Running
}

// GetFingerprint returns a phase fingerprint which is used to compare phases
func (p *Phase) GetFingerprint() Fingerprint {
	return Fingerprint{p.name, p.roundID}
}
