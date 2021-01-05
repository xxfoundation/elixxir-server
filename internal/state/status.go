///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////
package state

import (
	"fmt"
	"github.com/pkg/errors"
	"sync"
	"time"
)

type Status uint32

const (
	NOT_STARTED = Status(iota)
	STARTED
	ENDED
	NUM_STATUS
)

// Stringer to get the name of the activity, primarily for for error prints
func (s Status) String() string {
	switch s {
	case NOT_STARTED:
		return "NOT_STARTED"
	case STARTED:
		return "STARTED"
	case ENDED:
		return "ENDED"
	default:
		return fmt.Sprintf("UNKNOWN STATE: %d", s)
	}
}

//core state machine object
type GenericMachine struct {
	//holds the state
	*Status
	//mux to ensure proper access to state
	*sync.RWMutex

	//used to signal to waiting threads that a state change has occurred
	signal chan Status

	//changeChan
	changebuffer chan Status
}

//trinary, not started, started, ended
// make it waitForActive()
// makeActive() end()
// instantiate in instance
// make it generic,

func NewGenericMachine() GenericMachine {
	ss := NOT_STARTED

	//builds the object
	GM := GenericMachine{&ss,
		&sync.RWMutex{},
		make(chan Status),
		make(chan Status, 100),
	}

	return GM
}

func (gm GenericMachine) Start() error {
	_, err := gm.stateChange(*gm.Status)
	return err
}

func (gm *GenericMachine) WaitFor(timeout time.Duration, expected ...Status) (Status, error) {
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

	// todo: comment
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

	// unlock the read lock, allows state changes to take effect
	gm.RUnlock()

	// wait for the state change to happen
	err := <-done

	return *gm.Status, err
}

func (gm *GenericMachine) Update(nextState Status) (bool, error) {
	gm.Lock()
	defer gm.Unlock()

	//execute the state change
	success, err := gm.stateChange(nextState)
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

func (gm GenericMachine) stateChange(nextState Status) (bool, error) {
	*gm.Status = nextState

	select {
	case gm.changebuffer <- nextState:
	default:
		return false, errors.New("State change buffer full")
	}

	return true, nil
}
