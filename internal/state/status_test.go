////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package state

import (
	"testing"
	"time"
)

func TestNewGenericMachine(t *testing.T) {
	m := NewGenericMachine()

	// check the state pointer is properly initialized
	if m.Status == nil {
		t.Errorf("State pointer in state object should not be nil")
	}

	if *m.Status != NOT_STARTED {
		t.Errorf("State should be %s, is %s", NOT_STARTED, *m.Status)
	}

	// check the RWMutex has been created
	if m.RWMutex == nil {
		t.Errorf("State mutex should exist")
	}

	//check that the signal channel works properly
	// check the notify channel works properly
	go func() {
		m.signal <- STARTED
	}()

	timer := time.NewTimer(time.Millisecond)
	select {
	case received := <-m.signal:
		if received != STARTED {
			t.Errorf("Unexpected signal from test signal."+
				"\n\tExpected: %s"+
				"\n\tReceived: %s", STARTED, received)
		}
	case <-timer.C:
		t.Errorf("Should not have timed out on testing signal channel")
	}
}

//test that WaitFor returns immediately when the state is already correct
func TestGenericMachine_WaitFor(t *testing.T) {
	//create a new state
	m := NewGenericMachine()

	*m.Status = STARTED

	curActivity, err := m.WaitFor(time.Millisecond, STARTED)

	if curActivity != STARTED {
		t.Errorf("WaitFor() returned false when doing check on state" +
			" which is already true")
	}

	if err != nil {
		t.Errorf("WaitFor() returned error when doing check on state " +
			"which is already true")
	}
}

//tests when it takes time for the state to come
func TestGenericMachine_WaitForState(t *testing.T) {
	//create a new state
	m := NewGenericMachine()

	*m.Status = STARTED

	//create runner which after delay will send wait for state
	go func() {
		time.Sleep(10 * time.Millisecond)
		m.signal <- ENDED
	}()

	//run wait for state with longer timeout than delay in update
	_, err := m.WaitFor(100*time.Millisecond, ENDED)

	if *m.Status != STARTED {
		t.Errorf("WaitFor() returned true when doing check on state" +
			" which should have happened")
	}

	if err != nil {
		t.Errorf("WaitFor() returned an error when doing check on state" +
			" which should have happened correctly")
	}
}
