////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package state

import (
	"fmt"
	"github.com/pkg/errors"
	"sync"
	"time"
)

// This package holds the server's state object. It defines what states exist
// and what state transitions are allowable within the newState() function.
// Builds the state machiene documented in the cBetaNet document
// (https://docs.google.com/document/d/1qKeJVrerYmUmlwOgc2grhcS2Z4qdITcFB8xr49AGPKw/edit?usp=sharing)
//
// This should be used along side a business logic structure as follows:

/*
func main() {

	//run the state machiene
	for s := state.Get(); s!=CRASH;s = state.GetUpdate(){
		switch s{
		case NOT_STARTED:

		case WAITING:

		case PRECOMPUTING:

		case STANDBY:

		case REALTIME:

		case ERROR:
		}
	}

	//handle the crash state

}
 */

//state singleton
var s = newState()

/*///State Type///////////////////////////////////////////////////////////////*/

// type which holds states so they have have an associated stringer
type State uint8

// List of states server can be in
const(
	_ = State(iota)
	NOT_STARTED
	WAITING
	PRECOMPUTING
	STANDBY
	REALTIME
	ERROR
	CRASH
)

const NUM_STATES = CRASH + 1

// Stringer to get the name of the state
func (s State)String()string{
	switch(s){
	case NOT_STARTED: return "NOT_STARTED"
	case WAITING: return "WAITING"
	case PRECOMPUTING: return "PRECOMPUTING"
	case STANDBY: return "STANDBY"
	case REALTIME: return "REALTIME"
	case ERROR: return "ERROR"
	case CRASH: return "CRASH"
	default: return fmt.Sprintf("UNKNOWN STATE: %d", s)
	}
}

/*///State Object/////////////////////////////////////////////////////////////*/

//core state object
type stateObj struct{
	//holds the state
	*State
	//mux to ensure proper access to state
	*sync.RWMutex
	//used to notify of state updates
	notify chan State
	//holds valid state transitions
	stateMap [][]bool
}

// builds the stateObj  and sets valid transitions
func newState() stateObj {
	s := NOT_STARTED

	//builds the object
	S := stateObj{&s,
		&sync.RWMutex{},
		make(chan State),
		make([][]bool, NUM_STATES, NUM_STATES),
	}

	//add state transitions
	S.addStateTransition(NOT_STARTED,WAITING,CRASH)
	S.addStateTransition(WAITING,PRECOMPUTING,ERROR)
	S.addStateTransition(PRECOMPUTING,STANDBY,ERROR)
	S.addStateTransition(REALTIME,WAITING,ERROR)
	S.addStateTransition(ERROR,WAITING,CRASH)

	return S
}

// adds a state transition to the state object
func (s stateObj)addStateTransition(from State, to ...State){
	for _, t:=range(to){
		s.stateMap[from][t] = true
	}
}

/*///Public Functions/////////////////////////////////////////////////////////*/

// if the requested state update is valid from the current state, moves the
// next state and updates any go routines waiting on the state update.
// returns a boolean if the update cannot be done and an error explaining
// why
func Update(nextState State)(bool,error){
	s.Lock()
	defer  s.Unlock()
	// check if the requested state change is valid
	if !s.stateMap[*s.State][nextState] {
		// return an error if state change if invalid
		return false, errors.New("not a valid state change from %s to %s")
	}

	// set the state
	*s.State=nextState
	// notify the minimum of 1 waiting for transition within the business
	// logic loop in a blocking manner
	s.notify<-nextState
	// notify others until there are no more to notify by returning until there
	// are non waiting on the channel
	for notify:=true;notify;{
		select{
		case s.notify<- nextState:
		default:
			notify=false
		}
	}
	return true, nil

}

// gets the current state under a read lock
func Get()State{
	s.RLock()
	defer s.RUnlock()
	return *s.State
}

// waits to be notified and then returns an update. This should only be used
// once in the core state machine loop.
func GetUpdate()State{
	<-s.notify
	s.RLock()
	defer s.RUnlock()
	return *s.State
}

// if the the passed state is the next state update, waits until that update
// happens. return true if the waited state is the current state. returns an
// error after the timeout expires
func WaitFor(expected State, timeout time.Duration)(bool, error){
	// take the read lock to ensure state does not change during intital
	// checks
	s.RLock()

	// channels to control and receive from the worker thread
	kill := make(chan struct{})
	done := make(chan error)

	// start a thread to reserve a spot to get a notification on state updates
	// state updates cannot happen until the state read lock is released, so
	// this wont do anything until the initial checks are done, but will ensure
	// there are no laps in being ready to receive a notifications
	timer := time.NewTimer(timeout)
	go func(){
		// wait on a state change notification or a timeout
		select{
		case newState:=<-s.notify:
			if newState!= expected {
				done <- errors.Errorf("State not updated to the " +
					"correct state: expected: %s receive: %s", expected,
					newState)
			}else{
				done <- nil
			}
		case <-timer.C:
			done <- errors.Errorf("Timer of %s timed out before " +
				"state update", timeout)
		case <-kill:
		}
	}()

	// if already in the state return true
	if *s.State== expected {
		// kill the worker thread
		kill<-struct{}{}
		// release the read lock
		s.RUnlock()
		// return that the state is correct
		return true, nil
	}

	// if not in the state and the expected state cannot be reached from the
	// current one, return false and an error
	if !s.stateMap[*s.State][expected]{
		// kill the worker thread
		kill<-struct{}{}
		// release the read lock
		s.RUnlock()
		// return the error
		return false, errors.Errorf("Cannot wait for state %s which "+
			"cannot be gotten to from the current state %s", expected, *s.State)
	}

	// unlock the read lock, allows state changes to take effect
	s.RUnlock()

	// wait for the state change to happen
	err := <-done

	// return the result
	if err!=nil{
		return false, err
	}

	return true, nil
}
