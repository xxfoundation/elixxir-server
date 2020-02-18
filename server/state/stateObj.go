package state

import (
	"github.com/pkg/errors"
	"sync"
	"time"
)

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

// if the requested state update is valid from the current state, moves the
// next state and updates any go routines waiting on the state update.
// returns a boolean if the update cannot be done and an error explaining
// why
func (s stateObj)update(nextState State)(bool,error){
	s.Lock()
	defer  s.Unlock()
	//check if the requested state change is valid
	if s.stateMap[*s.State][nextState]{
		//set the state
		*s.State=nextState
		//notify the minimum of 1 waiting for transition within the buisness
		//logic loop
		s.notify<-nextState
		//notify others until there are no more to notify
		for notify:=true;notify;{
			select{
			case s.notify<- nextState:
			default:
				notify=false
			}
		}
		return true, nil
	}else{
		//return an error if state change if invalid
		return false, errors.New("not a valid state change from %s to %s")
	}
}

// adds a state transition to the state object
func (s stateObj)addStateTransition(from State, to ...State){
	for _, t:=range(to){
		s.stateMap[from][t] = true
	}
}

// gets the current state under a read lock
func (s stateObj)get()State{
	s.RLock()
	defer s.RUnlock()
	return *s.State
}

// waits to be notified and then returns an update
func (s stateObj)getUpdate()State{
	<-s.notify
	s.RLock()
	defer s.RUnlock()
	return *s.State
}

// if the the passed state is the next state update, waits until that update
// happens. return true if the waited state is the current state
func (s stateObj)waitOn(expected State, timeout time.Duration)(bool, error){
	s.RLock()

	//channels to control and receive from the worker thread
	kill := make(chan struct{})
	done := make(chan error)

	// start a thread to reserve a spot to get a notification on state updates
	// state updates cannot happen until the state read lock is released, so this
	// wont do anything until the initial checks are done, but will ensure there
	// is no laps in being ready to receive a notifications
	//create the timer
	timer := time.NewTimer(timeout)
	go func(){
		//wait on a state change notification or a timeout
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

	//if already in the state return true
	if *s.State== expected {
		//kill the worker thread
		kill<-struct{}{}
		//return true
		return true, nil
	}

	// if not in the state and the expected state cannot be reached from the
	// current one, return false and an error
	if !s.stateMap[*s.State][expected]{
		//kill the worker thread
		kill<-struct{}{}
		//return true
		return false, errors.Errorf("Cannot wait for state %s which "+
			"cannot be gotten to from the current state %s", expected, *s.State)
	}

	//unlock the read lock, allows state changes to take effect
	s.RUnlock()

	//wait for the state change to happen
	err := <-done

	//return the result
	if err!=nil{
		return false, err
	}

	return true, nil
}