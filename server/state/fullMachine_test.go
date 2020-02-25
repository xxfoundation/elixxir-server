package state_test

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/elixxir/server/server/state"
	"testing"
	"time"
)

// tests the happy path usage of the state machine code in its entirety within a
// mock up of the intended business logic loop
func TestMockBusinessLoop(t *testing.T) {

	var m state.Machine

	// build result tracker and expected results
	activityCount := make([]int, states.NUM_STATES)
	expectedActivity := []int{1, 16, 15, 14, 14, 2, 1}

	done := make(chan struct{})

	// wrapper for function used to change the state with logging. run in a
	// new go routine
	generalUpdate := func(from, to states.State) {
		//wait for calling state to be complete
		done, err := m.WaitFor(from, 5*time.Millisecond)
		if !done {
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
	var stateChanges [states.NUM_STATES]state.Change

	//NOT_STARTED state
	stateChanges[states.NOT_STARTED] = func(from states.State) error {
		activityCount[states.NOT_STARTED]++
		// move to next state
		go generalUpdate(states.NOT_STARTED, states.WAITING)
		return nil
	}

	//WAITING State
	stateChanges[states.WAITING] = func(from states.State) error {
		activityCount[states.WAITING]++
		// return an error if we have run the number of designated times
		if activityCount[states.WAITING] == expectedActivity[states.WAITING] {
			return errors.New("error from waiting")
		}

		// otherwise move to next state
		go generalUpdate(states.WAITING, states.PRECOMPUTING)

		return nil
	}

	//PRECOMPUTING State
	stateChanges[states.PRECOMPUTING] = func(from states.State) error {
		activityCount[states.PRECOMPUTING]++
		// return an error if we have run the number of designated times
		if activityCount[states.PRECOMPUTING] ==
			expectedActivity[states.PRECOMPUTING] {

			return errors.New("error from precomputing")
		}

		// otherwise move to next state
		go generalUpdate(states.PRECOMPUTING, states.STANDBY)

		return nil

	}

	//STANDBY State
	stateChanges[states.STANDBY] = func(from states.State) error {
		activityCount[states.STANDBY]++
		// move to next state
		go generalUpdate(states.STANDBY, states.REALTIME)

		return nil
	}

	//REALTIME State
	stateChanges[states.REALTIME] = func(from states.State) error {
		activityCount[states.REALTIME]++
		// move to next state
		go generalUpdate(states.REALTIME, states.WAITING)

		return nil
	}

	//ERROR State
	stateChanges[states.ERROR] = func(from states.State) error {
		activityCount[states.ERROR]++
		// return an error if we have run the number of designated times
		if activityCount[states.ERROR] == expectedActivity[states.ERROR] {
			// move to crash state
			go func() {
				done, err := m.WaitFor(from, 5*time.Millisecond)
				if !done {
					t.Logf("State %s never came: %s", from, err)
				}
				b, err := m.Update(states.CRASH)
				if !b {
					t.Errorf("Failure when updating to %s: %s",
						states.CRASH, err.Error())
				}
			}()
			// signal success
			return errors.New("crashing")
		} else {
			// move to next state
			go generalUpdate(states.ERROR, states.WAITING)
		}
		return nil
	}

	//CRASH State
	stateChanges[states.CRASH] = func(from states.State) error {
		activityCount[states.CRASH]++
		done <- struct{}{}
		return nil
	}

	m, err := state.NewMachine(stateChanges)

	//check if an error was returned
	if err != nil {
		t.Errorf("NewMachine() errored unexpectedly %s", err)
	}

	//wait for test to complete
	<-done

	// check that the final state is correct
	finalState := m.Get()
	if finalState != states.CRASH {
		t.Errorf("Final state not correct; expected: %s, recieved: %s",
			states.CRASH, finalState)
	}

	// check if the state machine executed properly. make sure each state was
	// executed the correct number of times
	for i := states.NOT_STARTED; i < states.NUM_STATES; i++ {
		if activityCount[i] != expectedActivity[i] {
			t.Errorf("State %s did not exicute enough times. "+
				"Exicuted %d times instead of %d", i, activityCount[i],
				expectedActivity[i])
		}
	}
}
