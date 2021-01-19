///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////
package state

import (
	"github.com/pkg/errors"
	"sync"
	"time"
)

//core state machine object
type GenericMachine struct {
	//holds the state
	*Status
	//mux to ensure proper access to state
	*sync.RWMutex

	//used to signal to waiting threads that a state change has occurred
	signal chan Status

	//holds valid state transitions
	stateMap [][]bool
}

// Constructor which generates a generic state machine
func NewGenericMachine() GenericMachine {
	ss := NOT_STARTED

	//builds the object
	GM := GenericMachine{&ss,
		&sync.RWMutex{},
		make(chan Status),
		make([][]bool, NUM_STATUS),
	}

	//finish populating the stateMap
	for i := 0; i < int(NUM_STATUS); i++ {
		GM.stateMap[i] = make([]bool, NUM_STATUS)
	}

	GM.addStateTransition(NOT_STARTED, STARTED)
	GM.addStateTransition(STARTED, ENDED)
	GM.addStateTransition(ENDED, NOT_STARTED)

	return GM
}

// Initiates the state machine
func (gm GenericMachine) Start() error {
	_, err := gm.stateChange(*gm.Status)
	return err
}

// if the the passed state is the next state update, waits until that update
// happens. return success if the waited state is the current state. returns an
// error after the timeout expires
func (gm GenericMachine) WaitFor(timeout time.Duration, expected ...Status) (Status, error) {
	// take the read lock to ensure state does not change during intital
	// checks
	gm.RLock()

	// channels to control and receive from the worker thread
	kill := make(chan struct{}, 1) // Size set to 1 to avoid race conditions
	done := make(chan error)

	// Place values in expected into a map
	expectedMap := make(map[Status]bool)
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
		case newState := <-gm.signal:
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
	if ok, _ := expectedMap[*gm.Status]; ok {
		// kill the worker thread
		kill <- struct{}{}
		// release the read lock
		gm.RUnlock()
		// return that the state is correct
		return *gm.Status, nil
	}

	validTransition := false

	// if not in the state and the expected state cannot be reached from the
	// current one, return false and an error
	for _, activity := range expected {
		if gm.stateMap[*gm.Status][activity] {
			validTransition = true
		}

	}

	if !validTransition {
		// kill the worker thread
		kill <- struct{}{}
		// release the read lock
		gm.RUnlock()
		// return the error
		return *gm.Status, errors.Errorf("Cannot wait for state %s which "+
			"cannot be reached from the current state %s", expected, *gm.Status)

	}

	// unlock the read lock, allows state changes to take effect
	gm.RUnlock()

	// wait for the state change to happen
	err := <-done

	return *gm.Status, err
}

// if the requested state update is valid from the current state, moves the
// next state and updates any go routines waiting on the state update.
// returns a boolean if the update cannot be done and an error explaining why
func (gm GenericMachine) Update(nextStatus Status) (bool, error) {
	gm.Lock()
	defer gm.Unlock()

	// check if the requested state change is valid
	if !gm.stateMap[*gm.Status][nextStatus] {
		// return an error if state change if invalid
		return false, errors.Errorf("not a valid state change from "+
			"%s to %s", *gm.Status, nextStatus)
	}

	//execute the state change
	success, err := gm.stateChange(nextStatus)
	if !success {
		return false, err
	}

	// notify threads waiting for state update until there are no more to notify by returning until there
	// are non waiting on the channel
	for signal := true; signal; {
		select {
		case gm.signal <- *gm.Status:
		default:
			signal = false
		}
	}

	return true, nil

}

// Wrapper around a call to update to not started, resetting the state machine
func (gm GenericMachine) Reset() (bool, error) {
	return gm.Update(NOT_STARTED)
}

// Internal function used to change states in NewGenericMachine() and Update()
func (gm GenericMachine) stateChange(nextState Status) (bool, error) {
	*gm.Status = nextState

	return true, nil
}

// adds a state transition to the state object
func (gm GenericMachine) addStateTransition(from Status, to ...Status) {
	for _, t := range to {
		gm.stateMap[from][t] = true
	}

}
