package phase

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/services"
	"sync"
	"sync/atomic"
	"time"
)

// Holds a single phase to be executed by the server in a round
type Phase struct {
	graph               *services.Graph
	tYpe                Type
	transmissionHandler Transmission
	timeout             time.Duration

	roundID    id.Round
	roundIDset sync.Once
	state      *uint32
	stateIndex int
	stateGroup *StateGroup
}

// New makes a new phase with the given graph, phase.Name, transmission handler, and timeout
func New(g *services.Graph, name Type, tHandler Transmission, timeout time.Duration) *Phase {
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
func (p *Phase) ConnectToRound(id id.Round, stateGroup *StateGroup) {
	p.roundIDset.Do(func() {
		p.roundID = id
		p.stateIndex, p.state = stateGroup.newState()
		p.stateGroup = stateGroup
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
	return State(atomic.LoadUint32(p.state))
}

// GetTransmissionHandler returns the phase's transmission handling function
func (p *Phase) GetTransmissionHandler() Transmission {
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
	return Fingerprint{p.tYpe, p.roundID}
}

// IncrementPhaseToQueued transitions Phase from Available to Queued
func (p *Phase) IncrementPhaseToQueued() bool {
	p.stateGroup.rw.RLock()
	defer p.stateGroup.rw.RUnlock()
	return atomic.CompareAndSwapUint32(p.state, uint32(Available), uint32(Queued))
}

// IncrementPhaseToQueued transitions Phase from Queued to Running
func (p *Phase) IncrementPhaseToRunning() bool {
	p.stateGroup.rw.RLock()
	defer p.stateGroup.rw.RUnlock()
	return atomic.CompareAndSwapUint32(p.state, uint32(Queued), uint32(Running))
}

// Finish transitions Phase from Running to Finished,
// and the phase after it from Initialized to Available
func (p *Phase) Finish() {
	p.stateGroup.rw.Lock()
	success := atomic.CompareAndSwapUint32(p.stateGroup.phase, uint32(p.tYpe), (uint32)(p.tYpe)+1)
	if !success {
		jww.FATAL.Panicf("Phase incremented incorrectly from %v as if %v in round %v",
			atomic.LoadUint32(p.state), p, p.roundID)
	}

	success = atomic.CompareAndSwapUint32(p.state, uint32(Running),
		uint32(Finished))
	if !success {
		jww.FATAL.Panicf("Phase state %v of running phase %s could not be"+
			" incremented to Finished", State(*p.state), p.tYpe.String())
	}

	if int(p.tYpe+1) < len(p.stateGroup.states) {
		success = atomic.CompareAndSwapUint32(p.stateGroup.states[p.stateIndex+1], uint32(Initialized), uint32(Available))
		if !success {
			jww.FATAL.Panicf("Phase state of new phase could not be incremented to Avalable")
		}
	}
	p.stateGroup.rw.Unlock()
}
