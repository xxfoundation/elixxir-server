package round

import (
	"fmt"
	"gitlab.com/elixxir/server/server/phase"
)

type ResponseMap map[string]Response

type Response struct {
	PhaseLookup    phase.Type
	ReturnPhase    phase.Type
	ExpectedStates []phase.State
}

func NewResponce(lookup, rtn phase.Type, expecteds ...phase.State) Response {
	return Response{PhaseLookup: lookup, ReturnPhase: rtn, ExpectedStates: expecteds}
}

func (r Response) CheckState(s phase.State) bool {
	for _, expected := range r.ExpectedStates {
		if s == expected {
			return true
		}
	}

	return false
}

func (r Response) String() string {
	validStates := " { "

	for _, s := range r.ExpectedStates[:len(r.ExpectedStates)-2] {
		validStates += s.String() + ", "
	}

	validStates += r.ExpectedStates[len(r.ExpectedStates)-1].String() + " } "

	return fmt.Sprintf("Phase.Responce{PhaseLookup: %s, ReturnPhase: %s, ExpectedStates: %s}",
		r.PhaseLookup, r.ReturnPhase, validStates)
}
