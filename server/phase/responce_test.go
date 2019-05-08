package phase

import (
	"reflect"
	"testing"
)

var lookup = Type(0)
var rtn = Type(1)
var expecteds = []State{2, 3, 4}

//Tests that the NewCMIXResponse returns the expected response
func TestNewCMIXResponse(t *testing.T) {
	rFace := NewCMIXResponse(lookup, rtn, expecteds...)

	rExpected := buildTestResponse()

	r := rFace.(CMixResponse)

	if !reflect.DeepEqual(r, rExpected) {
		t.Errorf("NewCMIXResponce: New Responce not the expected")
	}
}

//Checks that GetPhaseLookup returns the correct response
func TestCMixResponse_GetPhaseLookup(t *testing.T) {
	r := buildTestResponse()

	if r.GetPhaseLookup() != lookup {
		t.Errorf("CMixResponse.GetPhaseLookup: Expected: %s, Recieved:%s",
			lookup, r.GetPhaseLookup())
	}
}

//Checks that GetReturnPhase returns the correct response
func TestCMixResponse_GetReturnPhase(t *testing.T) {
	r := buildTestResponse()

	if r.GetReturnPhase() != rtn {
		t.Errorf("CMixResponse.GetReturnPhase: Expected: %s, Recieved:%s",
			rtn, r.GetReturnPhase())
	}
}

//Checks that CheckState returns true and false correctly
func TestCMixResponse_CheckState(t *testing.T) {
	r := buildTestResponse()

	if !r.CheckState(expecteds[0]) {
		t.Errorf("CMixResponse.CheckState: Returned false with valid state")
	}

	if r.CheckState(State(55)) {
		t.Errorf("CMixResponse.CheckState: Returned true with invalid state")
	}

}

//Checks that the stringer returns the correct string
func TestCMixResponse_String(t *testing.T) {
	r := buildTestResponse()

	expected := "phase.Responce{phaseLookup: 'PrecompGeneration', returnPhase:" +
		"'PrecompShare', expectedStates: {'Queued', 'Running', 'Computed'}}"

	if r.String() != expected {
		t.Error("CMixResponse.String: Did not return the correct string")
	}

}

//builds a response for testing
func buildTestResponse() CMixResponse {
	return CMixResponse{
		phaseLookup:    lookup,
		returnPhase:    rtn,
		expectedStates: expecteds,
	}
}
