package state

import (
	"errors"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"time"
)

// expected state transitions to be used in tests.  Should match the exact
// state transitions set in newState
var expectedStateMap = [][]bool{
[]bool{false, false, false, false, false, false, false, false},
[]bool{false, false, true, false, false, false, true, true},
[]bool{false, false, false, true, false, false, true, false},
[]bool{false, false, false, false, true, false, true, false},
[]bool{false, false, false, false, false, true, true, false},
[]bool{false, false, true, false, false, false, true, false},
[]bool{false, false, true, false, false, false, true, true},
[]bool{false, false, false, false, false, false, false, false},
}

//tests the test stringer is correct
func TestState_String(t *testing.T) {
	//define some states to check
	expectedStateStrings := []string{"UNKNOWN STATE: 0", "NOT_STARTED",
		"WAITING", "PRECOMPUTING", "STANDBY",  "REALTIME", "ERROR", "CRASH",
		"UNKNOWN STATE: 8"}

	//check if states give the correct return
	for st := State(0); st<=NUM_STATES;st++{
		if st.String() !=expectedStateStrings[st]{
			t.Errorf("State %d did not string correctly: expected: %s," +
				"recieved: %s", uint8(st), expectedStateStrings[st], st.String())
		}
	}
}

//tests the internal new state function creates a properly formed state object
func TestNewState(t *testing.T){
	so := newState()
	testNewStateObj(so, t)
}

//tests that the state singleton is created properly
func TestStateSingleton(t *testing.T){
	if reflect.DeepEqual(s,stateObj{}){
		t.Errorf("State singleton did no auto initialize")
	}
	testNewStateObj(s,t)
}

//function used to test that a state object has the correct initialization
func testNewStateObj(so stateObj, t *testing.T){
	// check the state pointer is properly initialized
	if so.State==nil{
		t.Errorf("State pointer in state object should not be nil")
	}

	if *so.State!=NOT_STARTED{
		t.Errorf("State should be %s, is %s", NOT_STARTED, *so.State)
	}

	// check the RWMutex has been created
	if so.RWMutex==nil{
		t.Errorf("State mutex should exist")
	}

	// check the notify channel works properly
	go func(){
		so.notify<- struct{f func(error); s State}{}
	}()

	timer := time.NewTimer(time.Millisecond)
	select{
		case <-so.notify:
		case <-timer.C:
			t.Errorf("Should not have timed out on testing notify channel")
	}

	//check that the signal channel works properly
	// check the notify channel works properly
	go func(){
		so.signal<- WAITING
	}()

	timer = time.NewTimer(time.Millisecond)
	select{
	case <-so.signal:
	case <-timer.C:
		t.Errorf("Should not have timed out on testing signal channel")
	}

	// check the initialized state map is correct
	if !reflect.DeepEqual(expectedStateMap,so.stateMap){
		t.Errorf("State map does not match expectated")
	}
}

//tests that state transitions are recorded properly
func TestAddStateTransition(t *testing.T){
	//do 100 random tests
	for i:=0;i<100;i++{
		//number of states each will transition to
		numStatesToo := uint8(rand.Uint64()%uint64(NUM_STATES-1))+1
		var stateList []State

		//generate states to transition to
		for j:=0;j<int(numStatesToo);j++{
			stateList =append(stateList, State(rand.Uint64()%uint64(NUM_STATES-1))+1)
		}

		for j:=State(1);j<NUM_STATES;j++{

			//build the object for the test
			S := stateObj{}
			S.stateMap = make([][]bool, NUM_STATES)

			for i:=0;i<int(NUM_STATES);i++{
				S.stateMap[i] = make([]bool, NUM_STATES)
			}

			//call addStateTransition
			S.addStateTransition(j,stateList...)

			//check that all states are correct
			for k:=State(0);k<NUM_STATES;k++{
				//find if k is in state list
				expected := false
				for _, st := range stateList{
					if st==k{
						expected = true
						break
					}
				}
				//check if the state is correct
				if S.stateMap[j][k]!=expected{
					t.Errorf("State was not as expected")
				}
			}
		}
	}
}

//test that all state transitions occur as expected
func TestUpdate_Transitions(t *testing.T) {
	s = newState()

	receivedStateTransition := make(chan struct{f func(error); s State})

	//create function to accept the notifications as transitions are iterated
	//through
	kill := make(chan struct{})
	go func(){
		for {
			select{
			case n := <-s.notify:
				n.f(nil)
				receivedStateTransition<-n
			case <-kill:
				break
			}
		}
	}()

	//test invalid state transitions
	for i:=State(0);i<NUM_STATES;i++{
		for j:=State(0);j<NUM_STATES;j++{
			*s.State = i
			success, err := Update(j)
			// if it is a valid state change make sure it is successful
			if expectedStateMap[i][j]{
				if !success || err!=nil{
					t.Errorf("Expected valid state transition from %s" +
						"to %s failed, err: %s", i, j,err)
				}
				// make sure the state transition was received and that it was
				// correct
				var received struct{f func(error); s State}

				timer := time.NewTimer(1*time.Millisecond)
				select{
				case received=<-receivedStateTransition:
				case <-timer.C:
					t.Errorf("Timed out on getting updateNotify from runner")
				}

				//check validity of receiver
				if received.f==nil{
					t.Errorf("Valid compeltion function not found in " +
						"Update Reicever")
				}

				if received.s != j{
					t.Errorf("State in update recever incorrect," +
						"Expected: %s, Recieved:%s", j, received.s)
				}

			// if it is an invalid state change make cure it fails and the
			// returns are correct
			}else{
				if success{
					t.Errorf("Expected invalid state transition from %s" +
						"to %s succeded, err: %s", i, j,err)
				}else if err ==nil {
					t.Errorf("Expected invalid state transition from %s" +
						"to %s failed but returned no error", i, j)
				}else if !strings.Contains(err.Error(),
						"not a valid state change from"){
					t.Errorf("Expected invalid state transition from %s" +
						"to %s failed with wrong error, err: %s", i, j,err)
				}
			}
		}
	}

	//kill the notification runner loop
	kill<- struct{}{}
}

//test state transition when the logic loop returns an error
func TestUpdate_TransitionError(t *testing.T) {
	s = newState()
	*s.State = PRECOMPUTING

	//create function to accept the notifications as transitions are iterated
	//through
	kill := make(chan struct{})
	go func(){
		//mock reception of state transition where an error is returned
		select{
		case n := <-s.notify:
			n.f(errors.New("mock error"))
		case <-kill:
			break
		}

		//mock reception of state transition where the error is handled
		select{
		case n := <-s.notify:
			n.f(nil)
		case <-kill:
			break
		}

	}()

	//try to update the state
	success, err := Update(STANDBY)
	if success{
		t.Errorf("Update succeded when it should have failed")
	}

	if err==nil{
		t.Errorf("Update should have returned an error, did not")
	}else if !strings.Contains(err.Error(),"mock error"){
		t.Errorf("Update returned wrong error, returned: %s", err.Error())
	}

}

//test state transition when the logic loop returns an error
func TestUpdate_TransitionDoubleError(t *testing.T) {
	s = newState()
	*s.State = PRECOMPUTING

	//create function to accept the notifications as transitions are iterated
	//through
	kill := make(chan struct{})
	go func(){
		//mock reception of state transition where an error is returned
		select{
		case n := <-s.notify:
			n.f(errors.New("mock error"))
		case <-kill:
			break
		}

		//mock reception of state transition where the error is handled
		select{
		case n := <-s.notify:
			n.f(errors.New("mock error2"))
		case <-kill:
			break
		}

	}()

	//try to update the state
	success, err := Update(STANDBY)
	if success{
		t.Errorf("Update succeded when it should have failed")
	}

	if err==nil{
		t.Errorf("Update should have returned an error, did not")
	}else if !strings.Contains(err.Error(),"mock error"){
		t.Errorf("Update returned wrong error, returned: %s", err.Error())
	}

}

//Test that all waiting channels get notified on update
func TestUpdate_ManyNotifications(t *testing.T) {
	numNotifications := 10
	timeout := 100*time.Millisecond

	s = newState()

	//create runner to clear the notification
	go func(){
		timer := time.NewTimer(5*time.Millisecond)
		select{
		case r := <- s.notify:
			r.f(nil)
		case <-timer.C:
			t.Errorf("Notification channel was never used")
		}
	}()

	//channel runners to be notified will return results on
	completion := make(chan bool)

	//function defining runners to be signaled
	notified := func(){
		timer := time.NewTimer(timeout)
		timedOut := false
		select{
			case st:=<-s.signal:
				if st!=WAITING{
					t.Errorf("signal runners recieved an update to "+
						"the wrong state: Expected: %s, Recieved: %s",
						WAITING, st)
				}
			case <-timer.C:
				timedOut = true
		}
		completion<-timedOut
	}

	//start all runners in their own go thread
	for i:=0;i<numNotifications;i++{
		go notified()
	}

	//wait so all runners start
	time.Sleep(1*time.Millisecond)

	//update to trigger the runners
	success, err := Update(WAITING)

	if !success || err!=nil{
		t.Errorf("Update that should have succeeded failed: ")
	}

	//check what happened to all runners
	numSuccess := 0
	numTimeout := 0
	for numSuccess+numTimeout<numNotifications{
		timedOut := <-completion
		if timedOut{
			numTimeout++
		}else{
			numSuccess++
		}
	}

	if numSuccess!=10{
		t.Errorf("%d runners did not get the update signal and timed " +
			"out", numTimeout)
	}
}

//test that get returns the correct value
func TestGet_Happy(t *testing.T) {
	numTest := 100
	for i:=0;i<numTest;i++{
		expectedState:= State(rand.Uint64()%uint64(NUM_STATES-1)+1)
		*s.State = expectedState
		recievedState:= Get()
		if recievedState!=expectedState{
			t.Errorf("Get returned the wrong value. " +
				"Expected: %v, Recieved: %s", expectedState, recievedState)
		}
	}
}

//test that get cannot return if the write lock is takes
func TestGet_Locked(t *testing.T) {

	//create a new state
	s=newState()
	*s.State = WAITING

	readState:= make(chan State)

	//lock the state so get cannot return
	s.Lock()

	//create a runner which polls get then returns the result over a channel
	go func(){
		st := Get()
		readState<-st
	}()

	//see if the state gets returned over the channel
	timer:=time.NewTimer(1*time.Millisecond)
	select{
		case <-readState:
			t.Errorf("Get() returned when it should be blocked")
		case <-timer.C:
	}

	//unlock the lock then check if the runner can read the state
	s.Unlock()

	timer=time.NewTimer(1*time.Millisecond)
	select{
	case st:=<-readState:
		if st!=WAITING{
			t.Errorf("Get() did not return the correct state. " +
				"Expected: %s, Recieved: %s", WAITING, st)
		}
	case <-timer.C:
		t.Errorf("Get() did not return when it should not have been " +
			"blocked")
	}
}

//tests that get update gets what is sent into the notify channel
func TestGetUpdate(t *testing.T) {
	s = newState()

	go func(){
		s.notify<-struct{f func(error); s State}{nil, WAITING}
	}()

	s, _ := GetUpdate()

	if s!=WAITING{
		t.Errorf("Did not recieve the correct state from GetUpdate()" +
			"Expected: %s, Recieved:L %s", WAITING, s)
	}
}

//test that WaitFor returns immediately when the state is already correct
func TestWaitFor_CorrectState(t *testing.T) {
	s = newState()

	*s.State = PRECOMPUTING

	b, err := WaitFor(PRECOMPUTING, time.Millisecond)

	if !b{
		t.Errorf("WaitFor() returned false when doing check on state" +
			" which is already true")
	}

	if err!=nil{
		t.Errorf("WaitFor() returned error when doing check on state " +
			"which is already true")
	}
}

//test that WaitFor returns an error when asked to wait for a state not
// reachable from the current
func TestWaitFor_Unreachable(t *testing.T) {
	s = newState()

	*s.State = PRECOMPUTING

	b, err := WaitFor(CRASH, time.Millisecond)

	if b{
		t.Errorf("WaitFor() succeded when the state cannot be reached")
	}

	if err==nil{
		t.Errorf("WaitFor() returned no error when the state "+
			"cannot be reached")
	}else if strings.Contains("cannot be reached from the current state",
		err.Error()){
		t.Errorf("WaitFor() returned the wrong error when the state "+
			"cannot be reached: %s", err.Error())
	}
}

//test the timeout for when the state change does not happen
func TestWaitFor_Timeout(t *testing.T) {
	s = newState()

	*s.State = PRECOMPUTING

	b, err := WaitFor(STANDBY, time.Millisecond)

	if b{
		t.Errorf("WaitFor() returned true when doing check on state" +
			" change which never happened")
	}

	if err==nil{
		t.Errorf("WaitFor() returned nil error when it should " +
			"have timed")
	}else if strings.Contains("timed out before state update", err.Error()){
		t.Errorf("WaitFor() returned the wrong error when timing out: %s",
			err)
	}
}

//tests when it takes time for the state to come
func TestWaitFor_WaitForState(t *testing.T) {
	s = newState()

	*s.State = PRECOMPUTING

	//create runner which after delay will send wait for state
	go func(){
		time.Sleep(10*time.Millisecond)
		s.signal<-STANDBY
	}()

	//run wait for state with longer timeout than delay in update
	b, err := WaitFor(STANDBY, 100*time.Millisecond)

	if!b{
		t.Errorf("WaitFor() returned true when doing check on state" +
			" which should have happened")
	}

	if err!=nil{
		t.Errorf("WaitFor() returned an error when doing check on state" +
			" which should have happened correctly")
	}
}

//tests when it takes time for the state to come
func TestWaitFor_WaitForBadState(t *testing.T) {
	s = newState()

	*s.State = PRECOMPUTING

	//create runner which after delay will send wait for state
	go func(){
		time.Sleep(10*time.Millisecond)
		s.signal<-ERROR
	}()

	//run wait for state with longer timeout than delay in update
	b, err := WaitFor(STANDBY, 100*time.Millisecond)

	if b{
		t.Errorf("WaitFor() returned true when doing check on state" +
			" transition which happened incorrectly")
	}

	if err==nil{
		t.Errorf("WaitFor() returned no error when bad state change "+
			"occured")
	}else if strings.Contains(err.Error(), "state not updated to the " +
	"correct state"){
		t.Errorf("WaitFor() returned thh wrong error on bad state " +
			"change: %s", err.Error())
	}
}

//test that Reset properly resets the state
func TestReset(t *testing.T) {
	*s.State = CRASH
	Reset(t)
	testNewStateObj(s,t)
}

//tests that Reset panics properly when it isn't passed a valid testing object
func TestReset_Panic(t *testing.T) {
	done := make(chan bool)

	go func(){
		defer func() {
			if r := recover(); r != nil {
				done<-true
			}
		}()
		Reset(nil)
		done<-false
	}()

	success:=<-done
	if !success{
		t.Errorf("Reset didnt panic when passed an invalid "+
			"testing object")
	}
}