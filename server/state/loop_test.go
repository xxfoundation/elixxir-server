package state_test

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/server/server/state"
	"testing"
)

func TestMockBusinessLoop(t *testing.T) {

	//reset the testing logic
	state.Reset(t)

	//build result tracker and expected results
	activityCount := make([]int, state.NUM_STATES)
	expectedActivity := []int{0,1,16,15,14,14,2,1}

	//run the state loop
	complete := func(error){}

	generalUpdate := func(st state.State){
		_, _= state.Update(st)
	}

	for s := state.Get(); s!=state.CRASH; s, complete = state.GetUpdate(){
		//increase the activity count to log what happened
		activityCount[s]++
		switch s{
		case state.NOT_STARTED:
			//signal state change is complete
			complete(nil)
			//move to next state
			go generalUpdate(state.WAITING)

		case state.WAITING:
			//return an error if we have run the number of designated times
			if activityCount[state.WAITING]==expectedActivity[state.WAITING]{
				complete(errors.New("error from waiting"))
			}else{
				//otherwise signal to move forward
				complete(nil)
				//move to next state
				go generalUpdate(state.PRECOMPUTING)
			}


		case state.PRECOMPUTING:
			//return an error if we have run the number of designated times
			if activityCount[state.PRECOMPUTING]==
				expectedActivity[state.PRECOMPUTING]{

				complete(errors.New("error from precomputing"))
			}else{
				//otherwise signal to move forward
				complete(nil)
				//move to next state
				go generalUpdate(state.STANDBY)
			}

		case state.STANDBY:
			//signal state change is complete
			complete(nil)
			//move to next state
			go generalUpdate(state.REALTIME)

		case state.REALTIME:

			//signal state change is complete
			complete(nil)
			//move to next state
			go generalUpdate(state.WAITING)

		case state.ERROR:
			//return an error if we have run the number of designated times
			if activityCount[state.ERROR]==expectedActivity[state.ERROR]{
				//signal success
				complete(errors.New("crashing"))
				//move to crash state
				go func(){
					b, err:= state.Update(state.CRASH)
					if !b{
						t.Errorf("Failure when updating to %s: %s",
							state.CRASH, err.Error())
					}
				}()
			}else if activityCount[state.ERROR]==
				expectedActivity[state.ERROR]-1{
				complete(nil)
				go generalUpdate(state.WAITING)
			}else{
				//otherwise signal to move forward
				complete(nil)
				//move to next state
				go generalUpdate(state.WAITING)
			}
		}
	}

	//complete the crash state
	complete(nil)

	activityCount[state.CRASH]++

	//check if the state machine executed properly
	for i:=state.NOT_STARTED;i<state.NUM_STATES;i++{
		if activityCount[i]!=expectedActivity[i]{
			t.Errorf("State %s did not exicute enough times. " +
				"Exicuted %d times instead of %d", i, activityCount[i],
				expectedActivity[i])
		}
	}
}
