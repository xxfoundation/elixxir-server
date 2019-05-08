package phase

import (
	"fmt"
)

// Response describes how the round should act when given an input coming
// from a specific phase.  The specific phase is looked up in the
// ResponseMap and the specifics are used to determine how to proceed
type ResponseMap map[string]Response

type Response interface {
	CheckState(State) bool
	GetPhaseLookup() Type
	GetReturnPhase() Type
	GetExpectedStates() []State
	fmt.Stringer
}

type CMixResponse struct {
	phaseLookup    Type
	returnPhase    Type
	expectedStates []State
}

//NewResponse Builds a new CMIX phase response adhering to the Response interface
func NewCMIXResponse(lookup, rtn Type, expecteds ...State) Response {
	return CMixResponse{phaseLookup: lookup, returnPhase: rtn, expectedStates: expecteds}
}

//GetPhaseLookup Returns the phaseLookup
func (r CMixResponse) GetPhaseLookup() Type {
	return r.phaseLookup
}

//GetReturnPhase returns the returnPhase
func (r CMixResponse) GetReturnPhase() Type {
	return r.returnPhase
}

//GetExpectedStates returns the expected states as a slice
func (r CMixResponse) GetExpectedStates() []State {
	return r.expectedStates
}

// CheckState returns true if the passed state is in
// the expected states list, otherwise it returns false
func (r CMixResponse) CheckState(state State) bool {
	for _, expected := range r.expectedStates {
		if state == expected {
			return true
		}
	}

	return false
}

// String adheres to the stringer interface
func (r CMixResponse) String() string {
	validStates := "{'"

	for _, s := range r.expectedStates[:len(r.expectedStates)-1] {
		validStates += s.String() + "', '"
	}

	validStates += r.expectedStates[len(r.expectedStates)-1].String() + "'}"

	return fmt.Sprintf("phase.Responce{phaseLookup: '%s', returnPhase:'%s', expectedStates: %s}",
		r.phaseLookup, r.returnPhase, validStates)
}
