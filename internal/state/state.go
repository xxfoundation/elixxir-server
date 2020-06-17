///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// This package holds the server's state object. It defines what states exist
// and what state transitions are allowable within the NewMachine() function.
// Builds the state machine documented in the cBetaNet document
// (https://docs.google.com/document/d/1qKeJVrerYmUmlwOgc2grhcS2Z4qdITcFB8xr49AGPKw/edit?usp=sharing)
//
// This requires a table of state conversion functions to function.
// An example implementation with state conversion functions can be found in state loop is implemented in loop_test.go and and example implementation
// business logic is as follows:

/*
func main() {

	var m state.Machine

	//create the state change function table
	var stateChanges [states.NUM_STATES]state.Change

	//NOT_STARTED state
	stateChanges[states.NOT_STARTED] = func(from states.State)error{
		// all the server startup code

		// signal state change is complete by returning,
		// returning an error if it failed
		return nil
	}

	//WAITING State
	stateChanges[states.WAITING] = func(from states.State)error{
		// start pre-precomputation

		// signal state change is complete by returning,
		// returning an error if it failed
		return nil
	}

	//PRECOMPUTING State
	stateChanges[states.PRECOMPUTING] = func(from states.State)error {

		// signal state change is complete by returning,
		// returning an error if it failed
		return nil
	}

	//STANDBY State
	stateChanges[states.STANDBY] = func(from states.State)error {
		// signal state change is complete by returning,
		// returning an error if it failed
		return nil
	}

	//REALTIME State
	stateChanges[states.REALTIME] = func(from states.State)error {
		// signal state change is complete by returning,
		// returning an error if it failed
		return nil
	}

	//ERROR State
	stateChanges[states.ERROR] = func(from states.State)error {
		// signal state change is complete by returning,
		// returning an error if it failed
		return nil
	}

	//CRASH State
	stateChanges[states.CRASH] = func(from states.State)error {
		// handle the crash
		panic()
		return nil
	}

	//start the state machine
	m = state.NewMachine(stateChanges)

	//block in main thread forever
	select{}
}
*/

package state

import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/primitives/current"
	"sync"
	"testing"
	"time"
)

/*///State Machine Object/////////////////////////////////////////////////////*/

// function which does a state change.  It should operate quickly, and cannot
// instruct state changes itself without creating a deadlock
type Change func(from current.Activity) error

//core state machine object
type Machine struct {
	//holds the state
	*current.Activity
	//mux to ensure proper access to state
	*sync.RWMutex
	//hold the functions used to change to different states
	changeList [current.NUM_STATES]Change

	//used to signal to waiting threads that a state change has occurred
	signal chan current.Activity
	//holds valid state transitions
	stateMap [][]bool
	//changeChan
	changebuffer chan current.Activity
}

func NewTestMachine(changeList [current.NUM_STATES]Change, start current.Activity, t interface{}) Machine {
	switch v := t.(type) {
	case *testing.T:
	case *testing.M:
		break
	default:
		panic(fmt.Sprintf("Cannot use outside of test environment; %+v", v))
	}

	m := NewMachine(changeList)
	*m.Activity = start

	return m
}

// builds the stateObj  and sets valid transitions
func NewMachine(changeList [current.NUM_STATES]Change) Machine {
	ss := current.NOT_STARTED

	//builds the object
	M := Machine{&ss,
		&sync.RWMutex{},
		changeList,
		make(chan current.Activity),
		make([][]bool, current.NUM_STATES),
		make(chan current.Activity, 100),
	}

	//finish populating the stateMap
	for i := 0; i < int(current.NUM_STATES); i++ {
		M.stateMap[i] = make([]bool, current.NUM_STATES)
	}

	//add state transitions
	M.addStateTransition(current.NOT_STARTED, current.WAITING, current.ERROR, current.CRASH)
	M.addStateTransition(current.WAITING, current.PRECOMPUTING, current.ERROR)
	M.addStateTransition(current.PRECOMPUTING, current.STANDBY, current.ERROR)
	M.addStateTransition(current.STANDBY, current.REALTIME, current.ERROR)
	M.addStateTransition(current.REALTIME, current.COMPLETED, current.ERROR)
	M.addStateTransition(current.COMPLETED, current.WAITING, current.ERROR)
	M.addStateTransition(current.ERROR, current.WAITING, current.ERROR, current.CRASH)

	return M
}

func (m Machine) Start() error {
	_, err := m.stateChange(*m.Activity)
	return err
}

// adds a state transition to the state object
func (m Machine) addStateTransition(from current.Activity, to ...current.Activity) {
	for _, t := range to {
		m.stateMap[from][t] = true
	}
}

/*///Public Functions/////////////////////////////////////////////////////////*/

// if the requested state update is valid from the current state, moves the
// next state and updates any go routines waiting on the state update.
// returns a boolean if the update cannot be done and an error explaining why
// UPDATE CANNOT BE CALLED WITHIN STATE CHANGE FUNCTIONS
func (m Machine) Update(nextState current.Activity) (bool, error) {
	m.Lock()
	defer m.Unlock()

	// Errors tend to cascade, so we should ignore attempts to transition into error from error
	if nextState == current.ERROR && *m.Activity == current.ERROR {
		return true, nil
	}

	jww.INFO.Printf("Updating to %v", nextState)

	// check if the requested state change is valid
	if !m.stateMap[*m.Activity][nextState] {
		// return an error if state change if invalid
		return false, errors.Errorf("not a valid state change from "+
			"%s to %s", *m.Activity, nextState)
	}

	//execute the state change
	success, err := m.stateChange(nextState)
	if !success {
		return false, err
	}

	// notify threads waiting for state update until there are no more to notify by returning until there
	// are non waiting on the channel
	for signal := true; signal; {
		select {
		case m.signal <- *m.Activity:
		default:
			signal = false
		}
	}
	return true, nil
}

// gets the current state under a read lock
func (m Machine) Get() current.Activity {
	m.RLock()
	defer m.RUnlock()
	return *m.Activity
}

// gets the current state under a read lock
func (m Machine) GetBuffer() <-chan current.Activity {
	m.RLock()
	defer m.RUnlock()
	return m.changebuffer
}

// if the the passed state is the next state update, waits until that update
// happens. return true if the waited state is the current state. returns an
// error after the timeout expires
func (m Machine) WaitFor(timeout time.Duration, expected ...current.Activity) (current.Activity, error) {
	// take the read lock to ensure state does not change during intital
	// checks
	m.RLock()

	// channels to control and receive from the worker thread
	kill := make(chan struct{})
	done := make(chan error)

	// Place values in expected into a map
	expectedMap := make(map[current.Activity]bool)
	for _, val := range expected {
		expectedMap[val] = true
	}

	// start a thread to reserve a spot to get a notification on state updates
	// state updates cannot happen until the state read lock is released, so
	// this wont do anything until the initial checks are done, but will ensure
	// there are no laps in being ready to receive a notifications
	timer := time.NewTimer(timeout)
	go func() {
		// wait on a state change notification or a timeout
		select {
		case newState := <-m.signal:
			if ok, _ := expectedMap[newState]; !ok {
				done <- errors.Errorf("State not updated to the "+
					"correct state: expected: %s receive: %s", expected,
					newState)
			} else {
				done <- nil
			}
		case <-timer.C:
			done <- errors.Errorf("Timer of %s timed out before "+
				"state update", timeout)
		case <-kill:
		}
	}()

	// if already in the state return true
	if ok, _ := expectedMap[*m.Activity]; ok {
		// kill the worker thread
		kill <- struct{}{}
		// release the read lock
		m.RUnlock()
		// return that the state is correct
		return *m.Activity, nil
	}

	validTransition := false

	// if not in the state and the expected state cannot be reached from the
	// current one, return false and an error
	for _, activity := range expected {
		if m.stateMap[*m.Activity][activity] {
			validTransition = true
		}

	}

	if !validTransition {
		// kill the worker thread
		kill <- struct{}{}
		// release the read lock
		m.RUnlock()
		// return the error
		return *m.Activity, errors.Errorf("Cannot wait for state %s which "+
			"cannot be reached from the current state %s", expected, *m.Activity)

	}

	// unlock the read lock, allows state changes to take effect
	m.RUnlock()

	// wait for the state change to happen
	err := <-done

	return *m.Activity, err
}

// Waits until an update to the given expected state happens.
// return true if the waited state is the current state. returns an
// error after the timeout expires.  Only for use in testing.
func (m Machine) WaitForUnsafe(expected current.Activity, timeout time.Duration,
	t *testing.T) (bool, error) {
	if t == nil {
		panic("cannot use WaitForUnsafe outside of tests")
	}
	// take the read lock to ensure state does not change during intital
	// checks
	m.RLock()

	// channels to control and receive from the worker thread
	kill := make(chan struct{})
	done := make(chan error)

	// start a thread to reserve a spot to get a notification on state updates
	// state updates cannot happen until the state read lock is released, so
	// this wont do anything until the initial checks are done, but will ensure
	// there are no laps in being ready to receive a notifications
	timer := time.NewTimer(timeout)
	go func() {
		// wait on a state change notification or a timeout
		select {
		case newState := <-m.signal:
			if newState != expected {
				done <- errors.Errorf("State not updated to the "+
					"correct state: expected: %s receive: %s", expected,
					newState)
			} else {
				done <- nil
			}
		case <-timer.C:
			done <- errors.Errorf("Timer of %s timed out before "+
				"state update", timeout)
		case <-kill:
		}
	}()

	// if already in the state return true
	if *m.Activity == expected {
		// kill the worker thread
		kill <- struct{}{}
		// release the read lock
		m.RUnlock()
		// return that the state is correct
		return true, nil
	}

	// unlock the read lock, allows state changes to take effect
	m.RUnlock()

	// wait for the state change to happen
	err := <-done

	// return the result
	if err != nil {
		return false, err
	}

	return true, nil
}

// Internal function used to change states in NewMachine() and Machine.Update()
func (m Machine) stateChange(nextState current.Activity) (bool, error) {
	oldState := *m.Activity
	*m.Activity = nextState

	select {
	case m.changebuffer <- nextState:
	default:
		return false, errors.New("State change buffer full")
	}

	err := m.changeList[*m.Activity](oldState)

	if err != nil {
		return false, err
	}

	return true, nil
}
