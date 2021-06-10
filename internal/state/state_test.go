///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package state

import (
	"errors"
	"fmt"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/current"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"time"
)

// expected state transitions to be used in tests.  Should match the exact
// state transitions set in newState
var expectedStateMap = [][]bool{
	{false, true, false, false, false, false, true, true},
	{false, false, true, false, false, false, true, false},
	{false, false, false, true, false, false, true, false},
	{false, false, false, false, true, false, true, false},
	{false, false, false, false, false, true, true, false},
	{false, true, false, false, false, false, true, false},
	{false, true, false, false, false, false, true, true},
	{false, false, false, false, false, false, false, false},
}

var dummyStates = [current.NUM_STATES]Change{
	func(from current.Activity, err *mixmessages.RoundError) error { return nil },
	func(from current.Activity, err *mixmessages.RoundError) error { return nil },
	func(from current.Activity, err *mixmessages.RoundError) error { return nil },
	func(from current.Activity, err *mixmessages.RoundError) error { return nil },
	func(from current.Activity, err *mixmessages.RoundError) error { return nil },
	func(from current.Activity, err *mixmessages.RoundError) error { return nil },
	func(from current.Activity, err *mixmessages.RoundError) error { return nil },
	func(from current.Activity, err *mixmessages.RoundError) error { return nil },
}

//tests that new Machiene works properly function creates a properly formed state object
func TestNewMachine(t *testing.T) {
	m := NewMachine(dummyStates)

	// check the state pointer is properly initialized
	if m.Activity == nil {
		t.Errorf("State pointer in state object should not be nil")
	}

	if *m.Activity != current.NOT_STARTED {
		t.Errorf("State should be %s, is %s", current.NOT_STARTED, *m.Activity)
	}

	// check the RWMutex has been created
	if m.RWMutex == nil {
		t.Errorf("State mutex should exist")
	}

	//check that the signal channel works properly
	// check the notify channel works properly
	go func() {
		m.signal <- current.WAITING
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
	for i := current.NOT_STARTED; i < current.NUM_STATES; i++ {
		if m.changeList[i] == nil {
			t.Errorf("Change function for %s is nil", i)
		}
	}
}

// tests that new Machine starts into error state when the passed NOT_STARTED
// change function errors
func TestNewMachine_Error(t *testing.T) {
	dummyStatesErr := dummyStates

	dummyStatesErr[current.NOT_STARTED] =
		func(from current.Activity, err *mixmessages.RoundError) error { return nil }

	dummyStatesErr[current.WAITING] =
		func(from current.Activity, err *mixmessages.RoundError) error { return errors.New("mock error") }

	m := NewMachine(dummyStatesErr)
	err := m.Start()
	if err != nil {
		t.Errorf("Failed to start state machine: %+v", err)
	}

	_, err = m.Update(current.WAITING)

	if err == nil {
		t.Error("Should have received an error")
	}

}

//tests that new Machiene works properly function creates a properly formed state object
func TestNewTestMachine(t *testing.T) {
	m := NewTestMachine(dummyStates, current.REALTIME, t)

	// check the state pointer is properly initialized
	if m.Activity == nil {
		t.Errorf("State pointer in state object should not be nil")
	}

	if *m.Activity != current.REALTIME {
		t.Errorf("State should be %s, is %s", current.NOT_STARTED, *m.Activity)
	}

	// check the RWMutex has been created
	if m.RWMutex == nil {
		t.Errorf("State mutex should exist")
	}

	//check that the signal channel works properly
	// check the notify channel works properly
	go func() {
		m.signal <- current.WAITING
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
	for i := current.NOT_STARTED; i < current.NUM_STATES; i++ {
		if m.changeList[i] == nil {
			t.Errorf("Change function for %s is nil", i)
		}
	}
}

//tests that new NewTestMachine panics when called without a testing object
func TestNewTestMachine_Panic(t *testing.T) {

	//catch the panic
	defer func() {
		if r := recover(); r != nil {
		}
	}()

	NewTestMachine(dummyStates, current.REALTIME, nil)

	//error if it somehow didnt panic
	t.Errorf("Panic did not occur in NewTestMachine")
}

//tests that state transitions are recorded properly
func TestAddStateTransition(t *testing.T) {
	//do 100 random tests
	for i := 0; i < 100; i++ {
		//number of states each will transition to
		numStatesToo := uint8(rand.Uint64()%uint64(current.NUM_STATES-1)) + 1
		var stateList []current.Activity

		//generate states to transition to
		for j := 0; j < int(numStatesToo); j++ {
			stateList = append(stateList,
				current.Activity(rand.Uint64()%uint64(current.NUM_STATES-1))+1)
		}

		for j := current.Activity(1); j < current.NUM_STATES; j++ {

			//build the object for the test
			M := Machine{}
			M.stateMap = make([][]bool, current.NUM_STATES)

			for i := 0; i < int(current.NUM_STATES); i++ {
				M.stateMap[i] = make([]bool, current.NUM_STATES)
			}

			//call addStateTransition
			M.addStateTransition(j, stateList...)

			//check that all states are correct
			for k := current.Activity(0); k < current.NUM_STATES; k++ {
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
	m := NewMachine(dummyStates)
	//test invalid state transitions
	for i := current.Activity(0); i < current.NUM_STATES; i++ {
		for j := current.Activity(0); j < current.NUM_STATES; j++ {
			fmt.Println(fmt.Sprintf("Testing transition from %s to %s", i.String(), j.String()))
			*m.Activity = i
			var success bool
			var err error
			if j == current.ERROR || j == current.CRASH {
				success, err = m.Update(j, &mixmessages.RoundError{})
			} else {
				success, err = m.Update(j)
			}
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

	dummyStatesErr[current.STANDBY] =
		func(from current.Activity, err *mixmessages.RoundError) error { return errors.New("mock error") }

	m := NewMachine(dummyStatesErr)

	*m.Activity = current.PRECOMPUTING

	//try to update the state
	success, err := m.Update(current.STANDBY)
	if success {
		t.Errorf("Update succeded when it should have failed")
	}

	if err == nil {
		t.Errorf("Update should have returned an error, did not")
	} else if !strings.Contains(err.Error(), "mock error") {
		t.Errorf("Update returned wrong error, returned: %s", err.Error())
	}

}

//Test that all waiting channels get notified on update
func TestUpdate_ManyNotifications(t *testing.T) {
	numNotifications := 10
	timeout := 100 * time.Millisecond

	m := NewMachine(dummyStates)

	//channel runners to be notified will return results on
	completion := make(chan bool)

	//function defining runners to be signaled
	notified := func() {
		timer := time.NewTimer(timeout)
		timedOut := false
		select {
		case st := <-m.signal:
			if st != current.WAITING {
				t.Errorf("signal runners received an update to "+
					"the wrong state: Expected: %s, Received: %s",
					current.WAITING, st)
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
	success, err := m.Update(current.WAITING)

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
	m := NewMachine(dummyStates)

	numTest := 100
	for i := 0; i < numTest; i++ {
		expectedState := current.Activity(rand.Uint64()%uint64(current.NUM_STATES-1) + 1)
		*m.Activity = expectedState
		receivedState := m.Get()
		if receivedState != expectedState {
			t.Errorf("Get returned the wrong value. "+
				"Expected: %v, Received: %s", expectedState, receivedState)
		}
	}
}

//test that get cannot return if the write lock is takes
func TestGet_Locked(t *testing.T) {

	//create a new state
	m := NewMachine(dummyStates)

	//set to waiting state
	*m.Activity = current.WAITING

	readState := make(chan current.Activity)

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
		if st != current.WAITING {
			t.Errorf("Get() did not return the correct state. "+
				"Expected: %s, Received: %s", current.WAITING, st)
		}
	case <-timer.C:
		t.Errorf("Get() did not return when it should not have been " +
			"blocked")
	}
}

//test that WaitFor returns immediately when the state is already correct
func TestWaitFor_CorrectState(t *testing.T) {
	//create a new state
	m := NewMachine(dummyStates)

	*m.Activity = current.PRECOMPUTING

	curActivity, err := m.WaitFor(time.Millisecond, current.PRECOMPUTING)

	if curActivity != current.PRECOMPUTING {
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
	m := NewMachine(dummyStates)

	*m.Activity = current.PRECOMPUTING

	curActivity, err := m.WaitFor(time.Millisecond, current.CRASH)

	if curActivity != current.PRECOMPUTING {
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
	m := NewMachine(dummyStates)

	*m.Activity = current.PRECOMPUTING

	curActivity, err := m.WaitFor(time.Millisecond, current.STANDBY)

	if curActivity != current.PRECOMPUTING {
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
	m := NewMachine(dummyStates)

	*m.Activity = current.PRECOMPUTING

	//create runner which after delay will send wait for state
	go func() {
		time.Sleep(10 * time.Millisecond)
		m.signal <- current.STANDBY
	}()

	//run wait for state with longer timeout than delay in update
	_, err := m.WaitFor(100*time.Millisecond, current.STANDBY)

	if *m.Activity != current.PRECOMPUTING {
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
	m := NewMachine(dummyStates)

	*m.Activity = current.PRECOMPUTING

	//create runner which after delay will send wait for state
	go func() {
		time.Sleep(10 * time.Millisecond)
		m.signal <- current.ERROR
	}()

	//run wait for state with longer timeout than delay in update
	curActivity, err := m.WaitFor(100*time.Millisecond, current.STANDBY)

	if curActivity != current.PRECOMPUTING {
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

//test that WaitForUnsafe returns immediately when the state is already correct
func TestWaitForUnsafe_CorrectState(t *testing.T) {
	//create a new state
	m := NewMachine(dummyStates)

	*m.Activity = current.PRECOMPUTING

	b, err := m.WaitForUnsafe(current.PRECOMPUTING, time.Millisecond, t)

	if !b {
		t.Errorf("WaitFor() returned false when doing check on state" +
			" which is already true")
	}

	if err != nil {
		t.Errorf("WaitFor() returned error when doing check on state " +
			"which is already true")
	}
}

//test the timeout for when the state change does not happen
func TestWaitForUnsafe_Timeout(t *testing.T) {
	//create a new state
	m := NewMachine(dummyStates)

	*m.Activity = current.PRECOMPUTING

	b, err := m.WaitForUnsafe(current.STANDBY, time.Millisecond, t)

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
func TestWaitForUnsafe_WaitForState(t *testing.T) {
	//create a new state
	m := NewMachine(dummyStates)

	*m.Activity = current.PRECOMPUTING

	//create runner which after delay will send wait for state
	go func() {
		time.Sleep(10 * time.Millisecond)
		m.signal <- current.STANDBY
	}()

	//run wait for state with longer timeout than delay in update
	b, err := m.WaitForUnsafe(current.STANDBY, 100*time.Millisecond, t)

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
func TestMachine_WaitForUnsafe_WaitForBadState(t *testing.T) {
	//create a new state
	m := NewMachine(dummyStates)

	*m.Activity = current.PRECOMPUTING

	//create runner which after delay will send wait for state
	go func() {
		time.Sleep(10 * time.Millisecond)
		m.signal <- current.ERROR
	}()

	//run wait for state with longer timeout than delay in update
	b, err := m.WaitForUnsafe(current.STANDBY, 100*time.Millisecond, t)

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

//tests when it takes time for the state to come
func TestMachine_WaitForUnsafe_Panic(t *testing.T) {
	//create a new state
	m := NewMachine(dummyStates)

	//catch the panic
	defer func() {
		if r := recover(); r != nil {
		}
	}()

	m.WaitForUnsafe(current.STANDBY, 100*time.Millisecond, nil)

	//error if it somehow didnt panic
	t.Errorf("Panic did not occur in WaitForUnsafe_Panic")
}
