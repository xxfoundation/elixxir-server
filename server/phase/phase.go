////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package phase

import (
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/services"
	"sync/atomic"
	"time"
)

//An interface which phase adheres to.  For use within
//Handler testing to allow the interface to be overwritten
type Phase interface {
	ConnectToRound(id id.Round, setState Transition,
		getState GetState)
	GetGraph() *services.Graph
	GetRoundID() id.Round
	GetType() Type
	GetTransmissionHandler() Transmit
	GetTimeout() time.Duration
	GetState() State
	UpdateFinalStates()
	AttemptToQueue(queue chan<- Phase) bool
	IsQueued() bool
	Send(chunk services.Chunk)
	Input(index uint32, slot *mixmessages.Slot) error
	Cmp(Phase) bool
	String() string
	Measure(tag string)
}

// Holds a single phase to be executed by the server in a round
type phase struct {
	graph               *services.Graph
	tYpe                Type
	transmissionHandler Transmit
	timeout             time.Duration

	roundID           id.Round
	connected         *uint32
	transitionToState Transition
	getState          GetState

	queued *uint32

	//This bool denotes if the phase goes straight to completed or waits for an
	//External check at Computed
	verification bool

	Metrics measure.Metrics
}

// New makes a new phase with the given the phase definition structure
// containing the graph, phase.Name, transmission handler, timeout, and
// verification flag
func New(def Definition) Phase {
	connected := uint32(0)
	queued := uint32(0)
	return &phase{
		graph:               def.Graph,
		tYpe:                def.Type,
		transmissionHandler: def.TransmissionHandler,
		timeout:             def.Timeout,
		verification:        def.DoVerification,
		connected:           &connected,
		queued:              &queued,
	}
}

/* Setters */

// ConnectToRound sets the round ID.  Can only be called once.
// Should only be called from Round package that initializes states
// Must be called on all phases in their order in the round
func (p *phase) ConnectToRound(id id.Round, setState Transition,
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
func (p *phase) GetGraph() *services.Graph {
	return p.graph
}

func (p *phase) GetRoundID() id.Round {
	return p.roundID
}

func (p *phase) GetType() Type {
	return p.tYpe
}

// GetState returns the current state of the phase
func (p *phase) GetState() State {
	return p.getState()
}

// AttemptToQueue attempts to set the internal phase state to queued.
// it returns success/failure. The phase will call itself on the passed
// function to queue itself
func (p *phase) AttemptToQueue(queue chan<- Phase) bool {
	success := atomic.CompareAndSwapUint32(p.queued, uint32(0), uint32(1))
	if success {
		queue <- p
	}
	return success
}

//IsQueued returns if the phase has been queued or not
func (p *phase) IsQueued() bool {
	return atomic.LoadUint32(p.queued) == 1
}

// UpdateFinalStates first transitions to the computed state and
// panics if unsuccessful. If the phase does not have verification,
// it then transitions to the verified state and panics if that
// fails. The function returns true if the final state is Verified,
// false otherwise.
// Fixme: find a better name that expresses it always moves towards
// finishing, but doesnt always finish, even when it returns false
// It it cannot move, it panics
func (p *phase) UpdateFinalStates() {

	if !p.verification {
		success := p.transitionToState(Active, Verified)

		if !success {
			jww.FATAL.Panicf("phase %s of round %v at incorrect state"+
				"to be transitioned to Computed", p.tYpe, p.roundID)
		}
	} else {
		success := p.transitionToState(Active, Computed)

		if !success {

			success = p.transitionToState(Computed, Verified)
			if !success {
				jww.FATAL.Panicf("phase %s of round %v at incorrect state"+
					"to be transitioned to Computed or Verified", p.tYpe, p.roundID)
			}
		}
	}
}

// GetTransmissionHandler returns the phase's transmission handling function
func (p *phase) GetTransmissionHandler() Transmit {
	return p.transmissionHandler
}

// GetTimeout gets the timeout at which the phase will fail
func (p *phase) GetTimeout() time.Duration {
	return p.timeout
}

/*Utility*/
// Cmp checks if two phases are the same
func (p *phase) Cmp(p2 Phase) bool {
	return p.roundID == p2.GetRoundID() && p.tYpe == p2.GetType()
}

//String adheres to the string interface
func (p *phase) String() string {
	return fmt.Sprintf("phase.phase{roundID: %v, phaseType: %s}",
		p.roundID, p.tYpe)
}

// Send via the graph. This function allows for this graph function
// to be accessed via the interface
func (p *phase) Send(chunk services.Chunk) {
	p.Measure("Sending a slot")
	p.graph.Send(chunk)

}

// Input updates the graph's stream with the passed data at the passed index
func (p *phase) Input(index uint32, slot *mixmessages.Slot) error {
	return p.GetGraph().GetStream().Input(index, slot)
}

// Measure logs the output of the measure function
func getMeasureInfo(p *phase, tag string) string {
	// Generate our metric and get the timestamp from it, plus a temp delta var
	timestamp := p.Metrics.Measure(tag)
	delta := time.Duration(0)

	// Calculate the difference between this event and the last one, if there is
	// a last one.
	if len(p.Metrics.Events) > 1 {
		prevTimestamp := p.Metrics.Events[len(p.Metrics.Events)-2].Timestamp
		currTimestamp := p.Metrics.Events[len(p.Metrics.Events)-1].Timestamp
		delta = currTimestamp.Sub(prevTimestamp)
	}

	// Format string to return
	result := fmt.Sprintf("Recorded phase measurement:\n\tround ID: %d\n\tphase: %d\n\t"+
		"tag: %s\n\ttimestamp: %s\n\tdelta: %s",
		p.roundID, p.tYpe, tag, timestamp.String(), delta.String())
	return result
}

// Wrapper function to log output to console
func (p *phase) Measure(tag string) {
	jww.DEBUG.Print(getMeasureInfo(p, tag))
}
