////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package phase

import (
	"fmt"
)

// Response defines how a round should handle an incoming communication
// That communication describes itself as being the completion on a phase
type ResponseMap map[string]Response

type Response interface {
	CheckState(State) bool
	GetPhaseLookup() Type
	GetExpectedStates() []State
	GetReturnPhase() Type
	fmt.Stringer
}

type ResponseDefinition struct {
	// PhaseAtSource is the phase a communication is the result of.
	PhaseAtSource Type
	// ExpectedStates are the states that phase must be in locally to proceed
	ExpectedStates []State
	// PhaseToExecute is the phase to execute locally
	PhaseToExecute Type
}

//NewResponse Builds a new CMIX phase ResponseDefinition adhering to the Response interface
func NewResponse(def ResponseDefinition) Response {
	return def.deepCopy()
}

//GetPhaseLookup Returns the PhaseAtSource
func (r ResponseDefinition) GetPhaseLookup() Type {
	return r.PhaseAtSource
}

//GetReturnPhase returns the PhaseToExecute
func (r ResponseDefinition) GetReturnPhase() Type {
	return r.PhaseToExecute
}

//GetExpectedStates returns the expected states as a slice
func (r ResponseDefinition) GetExpectedStates() []State {
	return r.ExpectedStates
}

// CheckState returns true if the passed state is in
// the expected states list, otherwise it returns false
func (r ResponseDefinition) CheckState(state State) bool {
	for _, expected := range r.ExpectedStates {
		if state == expected {
			return true
		}
	}

	return false
}

// String adheres to the stringer interface
func (r ResponseDefinition) String() string {
	validStates := "{'"

	for _, s := range r.ExpectedStates[:len(r.ExpectedStates)-1] {
		validStates += s.String() + "', '"
	}

	validStates += r.ExpectedStates[len(r.ExpectedStates)-1].String() + "'}"

	return fmt.Sprintf("phase.Responce{PhaseAtSource: '%s', PhaseToExecute:'%s', ExpectedStates: %s}",
		r.PhaseAtSource, r.PhaseToExecute, validStates)
}

//deepCopy Creates a deep copy of the ResponseDefinition
func (rd ResponseDefinition) deepCopy() ResponseDefinition {
	rdNew := ResponseDefinition{
		PhaseAtSource:  rd.PhaseAtSource,
		PhaseToExecute: rd.PhaseToExecute,
		ExpectedStates: make([]State, len(rd.ExpectedStates)),
	}

	copy(rdNew.ExpectedStates, rd.ExpectedStates)
	return rdNew
}
