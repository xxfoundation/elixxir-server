///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package phase

import (
	"reflect"
	"testing"
)

var lookup = Type(0)
var rtn = Type(1)
var expecteds = []State{1}

//Tests that the NewResponse returns the expected ResponseDefinition
func TestNewCMIXResponse(t *testing.T) {
	rFace := NewResponse(ResponseDefinition{lookup,
		expecteds, rtn})

	rExpected := buildTestResponse()

	r := rFace.(ResponseDefinition)

	if !reflect.DeepEqual(r, rExpected) {
		t.Errorf("NewCMIXResponce: New Responce not the expected")
	}
}

//Checks that GetPhaseLookup returns the correct ResponseDefinition
func TestCMixResponse_GetPhaseLookup(t *testing.T) {
	r := buildTestResponse()

	if r.GetPhaseLookup() != lookup {
		t.Errorf("ResponseDefinition.GetPhaseLookup: Expected: %s, Received:%s",
			lookup, r.GetPhaseLookup())
	}
}

//Checks that GetReturnPhase returns the correct ResponseDefinition
func TestCMixResponse_GetReturnPhase(t *testing.T) {
	r := buildTestResponse()

	if r.GetReturnPhase() != rtn {
		t.Errorf("ResponseDefinition.GetReturnPhase: Expected: %s, Received:%s",
			rtn, r.GetReturnPhase())
	}
}

//Checks that CheckState returns true and false correctly
func TestCMixResponse_CheckState(t *testing.T) {
	r := buildTestResponse()

	if !r.CheckState(expecteds[0]) {
		t.Errorf("ResponseDefinition.CheckState: Returned false with valid state")
	}

	if r.CheckState(State(55)) {
		t.Errorf("ResponseDefinition.CheckState: Returned true with invalid state")
	}

}

//Checks that the stringer returns the correct string
func TestCMixResponse_String(t *testing.T) {
	r := buildTestResponse()

	expected := "phase.Responce{PhaseAtSource: 'PrecompGeneration', PhaseToExecute:" +
		"'PrecompShare', ExpectedStates: {'Active'}}"

	if r.String() != expected {
		t.Error("ResponseDefinition.String: Did not return the correct string")
	}

}

//builds a ResponseDefinition for testing
func buildTestResponse() ResponseDefinition {
	return ResponseDefinition{
		PhaseAtSource:  lookup,
		PhaseToExecute: rtn,
		ExpectedStates: expecteds,
	}
}
