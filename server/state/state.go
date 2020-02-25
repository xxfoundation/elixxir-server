////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package state

import (
	"fmt"
	"github.com/pkg/errors"
	//jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/primitives/states"
	"sync"
	"time"
)

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

/*///State Machine Object/////////////////////////////////////////////////////*/

// function which does a state change.  It should operate quickly, and cannot
// instruct state changes itself without creating a deadlock
type Change func(from states.State) error

//core state machine object
type Machine struct {
	//holds the state
	*states.State
	//mux to ensure proper access to state
	*sync.RWMutex
	//hold the functions used to change to different states
	changeList [states.NUM_STATES]Change

	//used to signal to waiting threads that a state change has occurred
	signal chan states.State
	//holds valid state transitions
	stateMap [][]bool
}

// builds the stateObj  and sets valid transitions
func NewMachine(changeList [states.NUM_STATES]Change) (Machine, error) {
	ss := states.NOT_STARTED

	//builds the object
	M := Machine{&ss,
		&sync.RWMutex{},
		changeList,
		make(chan states.State),
		make([][]bool, states.NUM_STATES),
	}

	//finish populating the stateMap
	for i := 0; i < int(states.NUM_STATES); i++ {
		M.stateMap[i] = make([]bool, states.NUM_STATES)
	}

	//add state transitions
	M.addStateTransition(states.NOT_STARTED, states.WAITING, states.ERROR, states.CRASH)
	M.addStateTransition(states.WAITING, states.PRECOMPUTING, states.ERROR)
	M.addStateTransition(states.PRECOMPUTING, states.STANDBY, states.ERROR)
	M.addStateTransition(states.STANDBY, states.REALTIME, states.ERROR)
	M.addStateTransition(states.REALTIME, states.WAITING, states.ERROR)
	M.addStateTransition(states.ERROR, states.WAITING, states.ERROR, states.CRASH)

	//enter into NOT_STARTED State
	_, err := M.stateChange(states.NOT_STARTED)

	return M, err
}

// adds a state transition to the state object
func (m Machine) addStateTransition(from states.State, to ...states.State) {
	for _, t := range to {
		m.stateMap[from][t] = true
	}
}

/*///Public Functions/////////////////////////////////////////////////////////*/

// if the requested state update is valid from the current state, moves the
// next state and updates any go routines waiting on the state update.
// returns a boolean if the update cannot be done and an error explaining why
// UPDATE CANNOT BE CALLED WITHIN STATE CHANGE FUNCTIONS
func (m Machine) Update(nextState states.State) (bool, error) {
	m.Lock()
	defer m.Unlock()
	// check if the requested state change is valid
	if !m.stateMap[*m.State][nextState] {
		// return an error if state change if invalid
		return false, errors.Errorf("not a valid state change from "+
			"%s to %s", *m.State, nextState)
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
		case m.signal <- *m.State:
		default:
			signal = false
		}
	}
	return true, nil
}

// gets the current state under a read lock
func (m Machine) Get() states.State {
	m.RLock()
	defer m.RUnlock()
	return *m.State
}

// if the the passed state is the next state update, waits until that update
// happens. return true if the waited state is the current state. returns an
// error after the timeout expires
func (m Machine) WaitFor(expected states.State, timeout time.Duration) (bool, error) {
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
	if *m.State == expected {
		// kill the worker thread
		kill <- struct{}{}
		// release the read lock
		m.RUnlock()
		// return that the state is correct
		return true, nil
	}

	// if not in the state and the expected state cannot be reached from the
	// current one, return false and an error
	if !m.stateMap[*m.State][expected] {
		// kill the worker thread
		kill <- struct{}{}
		// release the read lock
		m.RUnlock()
		// return the error
		return false, errors.Errorf("Cannot wait for state %s which "+
			"cannot be reached from the current state %s", expected, *m.State)
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
func (m Machine) stateChange(nextState states.State) (bool, error) {
	oldState := *m.State
	*m.State = nextState
	err := m.changeList[*m.State](oldState)

	if err != nil {
		*m.State = states.ERROR
		var errState error

		//move to the error state if that was not the intention of the update call
		if nextState != states.ERROR {

			//wait for the error state to return
			errState = m.changeList[states.ERROR](oldState)
		}

		//return the error from the error state if it exists
		if errState == nil {
			err = errors.Wrap(err,
				fmt.Sprintf("Error occured on error state change from %s to %s,"+
					" moving to %s state", *m.State, nextState, states.ERROR))
		} else {
			err = errors.Wrap(err,
				fmt.Sprintf("Error occured on state change from %s to %s,"+
					" moving to %s state, error state returned: %s", *m.State,
					nextState, states.ERROR, errState.Error()))
		}
		return false, err
	}

	return true, nil
}
