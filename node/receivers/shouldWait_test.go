////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package receivers

import (
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/server/server/phase"
	"testing"
)

// A generic function for testing shouldwait, when passing in a phase it should return the expected activity
func testPhaseForActivity(p phase.Type, a current.Activity, t *testing.T) {
	expectedActivity := shouldWait(p)
	if expectedActivity != a {
		t.Logf("Phase %v not returning %v", p, a)
		t.Fail()
	}
}

// Test all the variables that should be returning precomputing
func TestShouldWait_ReturnPrecomputing(t *testing.T) {
	testPhaseForActivity(phase.PrecompDecrypt, current.PRECOMPUTING, t)
	testPhaseForActivity(phase.PrecompShare, current.PRECOMPUTING, t)
	testPhaseForActivity(phase.PrecompGeneration, current.PRECOMPUTING, t)
	testPhaseForActivity(phase.PrecompReveal, current.PRECOMPUTING, t)
	testPhaseForActivity(phase.PrecompPermute, current.PRECOMPUTING, t)
}

// Test all variables that should be returning REALTIME
func TestShouldWait_ReturnRealtime(t *testing.T) {
	testPhaseForActivity(phase.RealDecrypt, current.REALTIME, t)
	testPhaseForActivity(phase.RealPermute, current.REALTIME, t)
}

// Test all variables that should be returning ERRORS
func TestShouldWait_ReturnError(t *testing.T) {
	testPhaseForActivity(phase.Complete, current.ERROR, t)
	testPhaseForActivity(phase.PhaseError, current.ERROR, t)
}
