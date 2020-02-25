////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package state

import (
	"errors"
	"gitlab.com/elixxir/primitives/states"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"time"
)

// expected state transitions to be used in tests.  Should match the exact
// state transitions set in newState
var expectedStateMap = [][]bool{
	{false, true, false, false, false, true, true},
	{false, false, true, false, false, true, false},
	{false, false, false, true, false, true, false},
	{false, false, false, false, true, true, false},
	{false, true, false, false, false, true, false},
	{false, true, false, false, false, true, true},
	{false, false, false, false, false, false, false},
}

var dummyStates = [states.NUM_STATES]Change{
	func(from states.State) error { return nil },
	func(from states.State) error { return nil },
	func(from states.State) error { return nil },
	func(from states.State) error { return nil },
	func(from states.State) error { return nil },
	func(from states.State) error { return nil },
	func(from states.State) error { return nil },
}

//tests that new Machiene works properly function creates a properly formed state object
func TestNewMachine(t *testing.T) {
	m, err := NewMachine(dummyStates)

	//check if an error was returned
	if err != nil {
		t.Errorf("NewMachine() errored unexpectedly %s", err)
	}

	// check the state pointer is properly initialized
	if m.State == nil {
		t.Errorf("State pointer in state object should not be nil")
	}

	if *m.State != states.NOT_STARTED {
		t.Errorf("State should be %s, is %s", states.NOT_STARTED, *m.State)
	}

	// check the RWMutex has been created
	if m.RWMutex == nil {
		t.Errorf("State mutex should exist")
	}

	//check that the signal channel works properly
	// check the notify channel works properly
	go func() {
		m.signal <- states.WAITING
	}()

	timer := time.NewTimer(time.Millisecond)
	select {
	case <-m.signal:
	case <-timer.C:
		t.Errorf("Should not have timed out on testing signal channel")
	}

	// check the initialized state map is correct
	if !reflect.DeepEqual(expectedStateMap, m.stateMap) {
		t.Errorf("State map does not match expectated")
	}

	// check the change list is correct by checking if pointers are
	// correct
	for i := states.NOT_STARTED; i < states.NUM_STATES; i++ {
		if m.changeList[i] == nil {
			t.Errorf("Change function for %s is nil", i)
		}
	}
}

// tests that new Machine starts into error state when the passed NOT_STARTED
// change function errors
func TestNewMachine_Error(t *testing.T) {
	dummyStatesErr := dummyStates

	dummyStatesErr[states.NOT_STARTED] =
		func(from states.State) error { return errors.New("mock error") }

	m, err := NewMachine(dummyStatesErr)

	//check if an error was returned
	if err == nil {
		t.Errorf("NewMachine() did not error when expected")
	}

	if *m.State != states.ERROR {
		t.Errorf("NewMachine() did not enter %s state, entered %s",
			states.ERROR, *m.State)
	}

}

//tests that state transitions are recorded properly
func TestAddStateTransition(t *testing.T) {
	//do 100 random tests
	for i := 0; i < 100; i++ {
		//number of states each will transition to
		numStatesToo := uint8(rand.Uint64()%uint64(states.NUM_STATES-1)) + 1
		var stateList []states.State

		//generate states to transition to
		for j := 0; j < int(numStatesToo); j++ {
			stateList = append(stateList,
				states.State(rand.Uint64()%uint64(states.NUM_STATES-1))+1)
		}

		for j := states.State(1); j < states.NUM_STATES; j++ {

			//build the object for the test
			M := Machine{}
			M.stateMap = make([][]bool, states.NUM_STATES)

			for i := 0; i < int(states.NUM_STATES); i++ {
				M.stateMap[i] = make([]bool, states.NUM_STATES)
			}

			//call addStateTransition
			M.addStateTransition(j, stateList...)

			//check that all states are correct
			for k := states.State(0); k < states.NUM_STATES; k++ {
				//find if k is in state list
				expected := false
				for _, st := range stateList {
					if st == k {
						expected = true
						break
					}
				}
				//check if the state is correct
				if M.stateMap[j][k] != expected {
					t.Errorf("State was not as expected")
				}
			}
		}
	}
}

//test that all state transitions occur as expected
func TestUpdate_Transitions(t *testing.T) {
	m, err := NewMachine(dummyStates)

	//check if an error was returned
	if err != nil {
		t.Errorf("NewMachine() errored unexpectedly %s", err)
	}

	//test invalid state transitions
	for i := states.State(0); i < states.NUM_STATES; i++ {
		for j := states.State(0); j < states.NUM_STATES; j++ {
			*m.State = i
			success, err := m.Update(j)
			// if it is a valid state change make sure it is successful
			if expectedStateMap[i][j] {
				if !success || err != nil {
					t.Errorf("Expected valid state transition from %s"+
						"to %s failed, err: %s", i, j, err)
				}

				// if it is an invalid state change make cure it fails and the
				// returns are correct
			} else {
				if success {
					t.Errorf("Expected invalid state transition from %s"+
						"to %s succeded, err: %s", i, j, err)
				} else if err == nil {
					t.Errorf("Expected invalid state transition from %s"+
						"to %s failed but returned no error", i, j)
				} else if !strings.Contains(err.Error(),
					"not a valid state change from") {
					t.Errorf("Expected invalid state transition from %s"+
						"to %s failed with wrong error, err: %s", i, j, err)
				}
			}
		}
	}
}

//test state transition when the logic loop returns an error
func TestUpdate_TransitionError(t *testing.T) {
	dummyStatesErr := dummyStates

	dummyStatesErr[states.STANDBY] =
		func(from states.State) error { return errors.New("mock error") }

	m, err := NewMachine(dummyStatesErr)

	//check if an error was returned
	if err != nil {
		t.Errorf("NewMachine() errored unexpectedly %s", err)
	}

	*m.State = states.PRECOMPUTING

	//try to update the state
	success, err := m.Update(states.STANDBY)
	if success {
		t.Errorf("Update succeded when it should have failed")
	}

	if err == nil {
		t.Errorf("Update should have returned an error, did not")
	} else if !strings.Contains(err.Error(), "mock error") {
		t.Errorf("Update returned wrong error, returned: %s", err.Error())
	}

}

//test state transition when the logic loop returns an error
func TestUpdate_TransitionDoubleError(t *testing.T) {
	dummyStatesErr := dummyStates

	dummyStatesErr[states.STANDBY] =
		func(from states.State) error { return errors.New("mock error STANDBY") }
	dummyStatesErr[states.ERROR] =
		func(from states.State) error { return errors.New("mock error ERROR") }

	m, err := NewMachine(dummyStatesErr)

	if err != nil {
		t.Errorf("NewMachine() errored unexpectedly %s", err)
	}

	*m.State = states.PRECOMPUTING

	//try to update the state
	success, err := m.Update(states.STANDBY)
	if success {
		t.Errorf("Update succeded when it should have failed")
	}

	if err == nil {
		t.Errorf("Update should have returned an error, did not")
	} else if !strings.Contains(err.Error(), "mock error STANDBY") ||
		!strings.Contains(err.Error(), "mock error ERROR") {
		t.Errorf("Update returned wrong error, returned: %s", err.Error())
	}

}

//Test that all waiting channels get notified on update
func TestUpdate_ManyNotifications(t *testing.T) {
	numNotifications := 10
	timeout := 100 * time.Millisecond

	m, err := NewMachine(dummyStates)

	if err != nil {
		t.Errorf("NewMachine() errored unexpectedly %s", err)
	}

	//channel runners to be notified will return results on
	completion := make(chan bool)

	//function defining runners to be signaled
	notified := func() {
		timer := time.NewTimer(timeout)
		timedOut := false
		select {
		case st := <-m.signal:
			if st != states.WAITING {
				t.Errorf("signal runners recieved an update to "+
					"the wrong state: Expected: %s, Recieved: %s",
					states.WAITING, st)
			}
		case <-timer.C:
			timedOut = true
		}
		completion <- timedOut
	}

	//start all runners in their own go thread
	for i := 0; i < numNotifications; i++ {
		go notified()
	}

	//wait so all runners start
	time.Sleep(1 * time.Millisecond)

	//update to trigger the runners
	success, err := m.Update(states.WAITING)

	if !success || err != nil {
		t.Errorf("Update that should have succeeded failed: ")
	}

	//check what happened to all runners
	numSuccess := 0
	numTimeout := 0
	for numSuccess+numTimeout < numNotifications {
		timedOut := <-completion
		if timedOut {
			numTimeout++
		} else {
			numSuccess++
		}
	}

	if numSuccess != 10 {
		t.Errorf("%d runners did not get the update signal and timed "+
			"out", numTimeout)
	}
}

//test that get returns the correct value
func TestGet_Happy(t *testing.T) {
	m, err := NewMachine(dummyStates)

	if err != nil {
		t.Errorf("NewMachine() errored unexpectedly %s", err)
	}

	numTest := 100
	for i := 0; i < numTest; i++ {
		expectedState := states.State(rand.Uint64()%uint64(states.NUM_STATES-1) + 1)
		*m.State = expectedState
		recievedState := m.Get()
		if recievedState != expectedState {
			t.Errorf("Get returned the wrong value. "+
				"Expected: %v, Recieved: %s", expectedState, recievedState)
		}
	}
}

//test that get cannot return if the write lock is takes
func TestGet_Locked(t *testing.T) {

	//create a new state
	m, err := NewMachine(dummyStates)

	if err != nil {
		t.Errorf("NewMachine() errored unexpectedly %s", err)
	}

	//set to waiting state
	*m.State = states.WAITING

	readState := make(chan states.State)

	//lock the state so get cannot return
	m.Lock()

	//create a runner which polls get then returns the result over a channel
	go func() {
		st := m.Get()
		readState <- st
	}()

	//see if the state gets returned over the channel
	timer := time.NewTimer(1 * time.Millisecond)
	select {
	case <-readState:
		t.Errorf("Get() returned when it should be blocked")
	case <-timer.C:
	}

	//unlock the lock then check if the runner can read the state
	m.Unlock()

	timer = time.NewTimer(1 * time.Millisecond)
	select {
	case st := <-readState:
		if st != states.WAITING {
			t.Errorf("Get() did not return the correct state. "+
				"Expected: %s, Recieved: %s", states.WAITING, st)
		}
	case <-timer.C:
		t.Errorf("Get() did not return when it should not have been " +
			"blocked")
	}
}

//test that WaitFor returns immediately when the state is already correct
func TestWaitFor_CorrectState(t *testing.T) {
	//create a new state
	m, err := NewMachine(dummyStates)

	if err != nil {
		t.Errorf("NewMachine() errored unexpectedly %s", err)
	}

	*m.State = states.PRECOMPUTING

	b, err := m.WaitFor(states.PRECOMPUTING, time.Millisecond)

	if !b {
		t.Errorf("WaitFor() returned false when doing check on state" +
			" which is already true")
	}

	if err != nil {
		t.Errorf("WaitFor() returned error when doing check on state " +
			"which is already true")
	}
}

//test that WaitFor returns an error when asked to wait for a state not
// reachable from the current
func TestWaitFor_Unreachable(t *testing.T) {
	//create a new state
	m, err := NewMachine(dummyStates)

	if err != nil {
		t.Errorf("NewMachine() errored unexpectedly %s", err)
	}

	*m.State = states.PRECOMPUTING

	b, err := m.WaitFor(states.CRASH, time.Millisecond)

	if b {
		t.Errorf("WaitFor() succeded when the state cannot be reached")
	}

	if err == nil {
		t.Errorf("WaitFor() returned no error when the state " +
			"cannot be reached")
	} else if strings.Contains("cannot be reached from the current state",
		err.Error()) {
		t.Errorf("WaitFor() returned the wrong error when the state "+
			"cannot be reached: %s", err.Error())
	}
}

//test the timeout for when the state change does not happen
func TestWaitFor_Timeout(t *testing.T) {
	//create a new state
	m, err := NewMachine(dummyStates)

	if err != nil {
		t.Errorf("NewMachine() errored unexpectedly %s", err)
	}

	*m.State = states.PRECOMPUTING

	b, err := m.WaitFor(states.STANDBY, time.Millisecond)

	if b {
		t.Errorf("WaitFor() returned true when doing check on state" +
			" change which never happened")
	}

	if err == nil {
		t.Errorf("WaitFor() returned nil error when it should " +
			"have timed")
	} else if strings.Contains("timed out before state update", err.Error()) {
		t.Errorf("WaitFor() returned the wrong error when timing out: %s",
			err)
	}
}

//tests when it takes time for the state to come
func TestWaitFor_WaitForState(t *testing.T) {
	//create a new state
	m, err := NewMachine(dummyStates)

	if err != nil {
		t.Errorf("NewMachine() errored unexpectedly %s", err)
	}

	*m.State = states.PRECOMPUTING

	//create runner which after delay will send wait for state
	go func() {
		time.Sleep(10 * time.Millisecond)
		m.signal <- states.STANDBY
	}()

	//run wait for state with longer timeout than delay in update
	b, err := m.WaitFor(states.STANDBY, 100*time.Millisecond)

	if !b {
		t.Errorf("WaitFor() returned true when doing check on state" +
			" which should have happened")
	}

	if err != nil {
		t.Errorf("WaitFor() returned an error when doing check on state" +
			" which should have happened correctly")
	}
}

//tests when it takes time for the state to come
func TestWaitFor_WaitForBadState(t *testing.T) {
	//create a new state
	m, err := NewMachine(dummyStates)

	if err != nil {
		t.Errorf("NewMachine() errored unexpectedly %s", err)
	}

	*m.State = states.PRECOMPUTING

	//create runner which after delay will send wait for state
	go func() {
		time.Sleep(10 * time.Millisecond)
		m.signal <- states.ERROR
	}()

	//run wait for state with longer timeout than delay in update
	b, err := m.WaitFor(states.STANDBY, 100*time.Millisecond)

	if b {
		t.Errorf("WaitFor() returned true when doing check on state" +
			" transition which happened incorrectly")
	}

	if err == nil {
		t.Errorf("WaitFor() returned no error when bad state change " +
			"occured")
	} else if strings.Contains(err.Error(), "state not updated to the "+
		"correct state") {
		t.Errorf("WaitFor() returned thh wrong error on bad state "+
			"change: %s", err.Error())
	}
}
