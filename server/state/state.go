////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package state

import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"sync"
	"testing"
	"time"
)

// This package holds the server's state object. It defines what states exist
// and what state transitions are allowable within the newState() function.
// Builds the state machiene documented in the cBetaNet document
// (https://docs.google.com/document/d/1qKeJVrerYmUmlwOgc2grhcS2Z4qdITcFB8xr49AGPKw/edit?usp=sharing)
//
// This requires a state loop to function, otherwise Update calls will deadlock
// the state loop is implemented in loop_test.go and and example implementation
// business logic is as follows:

/*
func main() {

	//run the state loop
	complete := func(error){}
	for s := state.Get(); s!=state.CRASH; s, complete := state.GetUpdate(){
		switch s{
		case state.NOT_STARTED:
			//all the server startup code

			//signal state change is complete, returning an error if it failed
			complete(err)

		case state.WAITING:
			//start pre-precomputation
			//signal state change is complete, returning an error if it failed
			complete(err)

		case state.PRECOMPUTING:
			//create round
			//set round to "active"
			//kick off if first node
			//signal state change is complete, returning an error if it failed
			complete(err)

		case state.STANDBY:
			//start pre-precomputation
			//signal state change is complete, returning an error if it failed
			complete(err)

		case state.REALTIME:
			//set to active round
			//kick off if first node
			//start pre-precomputation
			//signal state change is complete, returning an error if it failed
			complete(err)

		case state.ERROR:
			//determine if we should crash or go to wait
			//wait until reported to premissioning
			//signal state change is complete, returning an error if it failed
			//error state should return an error if it will not recover
			complete(err)
		}
	}

	//handle the crash state
	panic()

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

// Stringer to get the name of the state, primarily for for error prints
func (s State)String()string{
	switch s {
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
	//used to notify the core business logic thread that a state update is
	//being attempted
	notify chan struct{f func(error); s State}
	//used to signal to waiting threads that a state change has occurred
	signal chan State
	//holds valid state transitions
	stateMap [][]bool
}

// builds the stateObj  and sets valid transitions
func newState() stateObj {
	ss := NOT_STARTED

	//builds the object
	S := stateObj{&ss,
		&sync.RWMutex{},
		make(chan struct{f func(error); s State}),
		make(chan State),
		make([][]bool, NUM_STATES),
	}

	//finish populating the stateMap
	for i:=0;i<int(NUM_STATES);i++{
		S.stateMap[i] = make([]bool, NUM_STATES)
	}

	//add state transitions
	S.addStateTransition(NOT_STARTED,WAITING,ERROR,CRASH)
	S.addStateTransition(WAITING,PRECOMPUTING,ERROR)
	S.addStateTransition(PRECOMPUTING,STANDBY,ERROR)
	S.addStateTransition(STANDBY,REALTIME,ERROR)
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
// returns a boolean if the update cannot be done and an error explaining why
// UPDATE CANNOT BE CALLED WITHIN THE STATE LOOP.
func Update(nextState State)(bool,error){
	s.Lock()
	defer  s.Unlock()
	// check if the requested state change is valid
	if !s.stateMap[*s.State][nextState] {
		// return an error if state change if invalid
		return false, errors.Errorf("not a valid state change from " +
			"%s to %s", *s.State,nextState)
	}

	// notify the state loop of the change and wait for its response
	errChan := make(chan error)
	errFunc:=func(e error){
		errChan<-e
	}
	s.notify<-struct{f func(error); s State}{errFunc, nextState}

	//set the state to the next state
	*s.State = nextState

	//wait for it to complete the state change
	err := <-errChan

	// if the state change produced an error, change to the error state and
	// send the state change signal to the buisness logic loop
	if err!=nil{
		*s.State = ERROR
		var errState error

		//move to the error state if that was not the intention of the update call
		if nextState!=ERROR{
			s.notify<-struct{f func(error); s State}{errFunc, ERROR}

			//wait for the error state to return
			errState = <-errChan
		}

		//return the error from the error state if it exists
		if errState==nil{
			err = errors.Wrap(err,
				fmt.Sprintf("Error occured on error state change from %s to %s," +
					" moving to %s state", *s.State, nextState, ERROR))
		}else{
			err = errors.Wrap(err,
				fmt.Sprintf("Error occured on state change from %s to %s," +
					" moving to %s state, error state returned: %s", *s.State,
					nextState, ERROR, errState.Error()))
		}

		return false, err
	}

	// notify threads waiting for state update until there are no more to notify by returning until there
	// are non waiting on the channel
	for signal:=true;signal;{
		select{
		case s.signal<- *s.State:
		default:
			signal=false
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
// DO NOT USE OUTSIDE OF STATE LOOP
func GetUpdate()(State, func(error)){
	sc:=<-s.notify
	return sc.s, sc.f
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
		case newState:=<-s.signal:
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
			"cannot be reached from the current state %s", expected, *s.State)
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

// resets the state object for external tests so they can be sure they are
// starting fresh
func Reset(t *testing.T){
	if t==nil{
		jww.FATAL.Panicf("state.Reset() is only valid within" +
			" testing infrastructure")
	}

	s = newState()
}
