////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package state_test

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/server/internals/state"
	"testing"
	"time"
)

// tests the happy path usage of the state machine code in its entirety within a
// mock up of the intended business logic loop
func TestMockBusinessLoop(t *testing.T) {

	var m state.Machine

	// build result tracker and expected results
	activityCount := make([]int, current.NUM_STATES)
	expectedActivity := []int{1, 16, 15, 14, 14, 14, 2, 1}

	done := make(chan struct{})

	// wrapper for function used to change the state with logging. run in a
	// new go routine
	generalUpdate := func(from, to current.Activity) {
		//wait for calling state to be complete
		curActivity, err := m.WaitFor(5*time.Millisecond, from)
		if curActivity != from {
			t.Logf("State %s never came: %s", from, err)
		}
		//move to next state
		b, err := m.Update(to)
		if !b {
			t.Logf("State update to %s from %s returned error: %s", to,
				from, err.Error())
		}
	}

	//create the state change function table
	var stateChanges [current.NUM_STATES]state.Change

	//NOT_STARTED state
	stateChanges[current.NOT_STARTED] = func(from current.Activity) error {
		activityCount[current.NOT_STARTED]++
		// move to next state
		go generalUpdate(current.NOT_STARTED, current.WAITING)
		return nil
	}

	//WAITING State
	stateChanges[current.WAITING] = func(from current.Activity) error {
		activityCount[current.WAITING]++
		// return an error if we have run the number of designated times
		if activityCount[current.WAITING] == expectedActivity[current.WAITING] {
			return errors.New("error from waiting")
		}

		// otherwise move to next state
		go generalUpdate(current.WAITING, current.PRECOMPUTING)

		return nil
	}

	//PRECOMPUTING State
	stateChanges[current.PRECOMPUTING] = func(from current.Activity) error {
		activityCount[current.PRECOMPUTING]++
		// return an error if we have run the number of designated times
		if activityCount[current.PRECOMPUTING] ==
			expectedActivity[current.PRECOMPUTING] {

			return errors.New("error from precomputing")
		}

		// otherwise move to next state
		go generalUpdate(current.PRECOMPUTING, current.STANDBY)

		return nil

	}

	//STANDBY State
	stateChanges[current.STANDBY] = func(from current.Activity) error {
		activityCount[current.STANDBY]++
		// move to next state
		go generalUpdate(current.STANDBY, current.REALTIME)

		return nil
	}

	//REALTIME State
	stateChanges[current.REALTIME] = func(from current.Activity) error {
		activityCount[current.REALTIME]++
		// move to next state
		go generalUpdate(current.REALTIME, current.COMPLETED)

		return nil
	}

	//COMPLETED State
	stateChanges[current.COMPLETED] = func(from current.Activity) error {
		activityCount[current.COMPLETED]++
		// move to next state
		go generalUpdate(current.COMPLETED, current.WAITING)

		return nil
	}

	//ERROR State
	stateChanges[current.ERROR] = func(from current.Activity) error {
		activityCount[current.ERROR]++
		// return an error if we have run the number of designated times
		if activityCount[current.ERROR] == expectedActivity[current.ERROR] {
			// move to crash state
			go func() {
				curActivity, err := m.WaitFor(5*time.Millisecond, from)
				if curActivity != from {
					t.Logf("State %s never came: %s", from, err)
				}
				b, err := m.Update(current.CRASH)
				if !b {
					t.Errorf("Failure when updating to %s: %s",
						current.CRASH, err.Error())
				}
			}()
			// signal success
			return errors.New("crashing")
		} else {
			// move to next state
			go generalUpdate(current.ERROR, current.WAITING)
		}
		return nil
	}

	//CRASH State
	stateChanges[current.CRASH] = func(from current.Activity) error {
		activityCount[current.CRASH]++
		done <- struct{}{}
		return nil
	}

	m = state.NewMachine(stateChanges)
	err := m.Start()
	//check if an error was returned
	if err != nil {
		t.Errorf("NewMachine() errored unexpectedly %s", err)
	}

	//wait for test to complete
	<-done

	// check that the final state is correct
	finalState := m.Get()
	if finalState != current.CRASH {
		t.Errorf("Final state not correct; expected: %s, recieved: %s",
			current.CRASH, finalState)
	}

	// check if the state machine executed properly. make sure each state was
	// executed the correct number of times
	for i := current.NOT_STARTED; i < current.NUM_STATES; i++ {
		if activityCount[i] != expectedActivity[i] {
			t.Errorf("State %s did not exicute enough times. "+
				"Exicuted %d times instead of %d", i, activityCount[i],
				expectedActivity[i])
		}
	}
}
