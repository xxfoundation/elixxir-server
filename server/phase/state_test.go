package phase

import (
	"fmt"
	"testing"
)

// Proves that a single phase can be taken through all of its states
// using the provided utility methods
func TestPhaseState_EnabledVerification(t *testing.T) {
	state := uint32(Initialized)

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("PhaseState: Unexpected panic on state %s: %+v", State(state), r)
		}
	}()

	// Real implementations would use atomics for all of this for better
	// thread safety.
	// However, testing threading functionality is outside of the scope of this
	// test, so these testing implementations don't use atomics for readability.
	// Do NOT create a real implementation without atomics!
	p := phase{
		transitionToState: func(from, to State) bool {
			// Make sure the state is the one after
			if from >= to {
				return false
			} else {
				if State(state) != from {
					return false
				}
				state = uint32(to)
			}
			return true
		},
		getState: func() State {
			return State(state)
		},
		connected:    new(uint32),
		verification: true,
	}

	state = uint32(Available)
	expected := Available
	if p.GetState() != Available {
		t.Errorf("State was %v, but should have been %v",
			p.GetState(), expected)
	}
	p.AttemptTransitionToQueued()
	expected = Queued
	if p.GetState() != expected {
		t.Errorf("State was %v, but should have been %v",
			p.GetState(), expected)
	}
	p.TransitionToRunning()
	expected = Running
	if p.GetState() != expected {
		t.Errorf("State was %v, but should have been %v",
			p.GetState(), expected)
	}

	p.UpdateFinalStates()
	expected = Computed
	if p.GetState() != expected {
		t.Errorf("State was %v, but should have been %v",
			p.GetState(), expected)
	}

	p.UpdateFinalStates()
	expected = Verified
	if p.GetState() != expected {
		t.Errorf("State was %v, but should have been %v",
			p.GetState(), expected)
	}
}

// Proves that a single phase can be taken through all of its states
// using the provided utility methods
func TestPhaseState_WithoutVerification(t *testing.T) {
	state := uint32(Initialized)

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("PhaseState: Unexpected panic on state %s: %+v", State(state), r)
		}
	}()
	// Real implementations would use atomics for all of this for better
	// thread safety.
	// However, testing threading functionality is outside of the scope of this
	// test, so these testing implementations don't use atomics for readability.
	// Do NOT create a real implementation without atomics!
	p := phase{
		transitionToState: func(from, to State) bool {
			// Make sure the state is the one after
			if from >= to {
				return false
			} else {
				if State(state) != from {
					return false
				}
				state = uint32(to)
			}
			return true
		},
		getState: func() State {
			return State(state)
		},
		connected: new(uint32),
	}

	state = uint32(Available)
	expected := Available
	if p.GetState() != Available {
		t.Errorf("State was %v, but should have been %v",
			p.GetState(), expected)
	}
	p.AttemptTransitionToQueued()
	expected = Queued
	if p.GetState() != expected {
		t.Errorf("State was %v, but should have been %v",
			p.GetState(), expected)
	}
	p.TransitionToRunning()
	expected = Running
	if p.GetState() != expected {
		t.Errorf("State was %v, but should have been %v",
			p.GetState(), expected)
	}
	p.UpdateFinalStates()
	expected = Verified
	if p.GetState() != expected {
		t.Errorf("State was %v, but should have been %v",
			p.GetState(), expected)
	}
}

func TestState_String(t *testing.T) {
	for state := Initialized; state < NumStates; state++ {
		if state.String() != stateStrings[state] {
			t.Errorf("State string %v didn't match %v at index %v",
				state.String(), stateStrings[state], uint32(state))
		}
	}
	if len(stateStrings) != int(NumStates) {
		t.Error("There aren't enough state strings")
	}
}
