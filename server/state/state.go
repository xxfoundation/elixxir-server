package state

import "time"

// This package holds the server's state object. It defines what states exist
// and what state transitions are allowable within the newState() function.
// Builds the state machiene documented in the cBetaNet document
// (https://docs.google.com/document/d/1qKeJVrerYmUmlwOgc2grhcS2Z4qdITcFB8xr49AGPKw/edit?usp=sharing)
//
// This should be used along side a business logic stricture as follows:

/*
func main() {

	//run the state machiene
	for s := Get(); s!=CRASH;s = GetUpdate(){
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

// underlying state names
var stateNames = []string{"NOT_STARTED", "WAITING", "PRECOMPUTING", "STANDBY",
	"REALTIME", "ERROR", "CRASH"}

// type which holds states so they have have an assoceated stringer
type State uint8

// List of states server can be in
const(
	NOT_STARTED = State(iota)
	WAITING
	PRECOMPUTING
	STANDBY
	REALTIME
	ERROR
	CRASH
)

// Stringer to get the name of the state
func (s State)String()string{
	if s>CRASH{
		return "UNKNOWN STATE"
	}

	return stateNames[s]
}

const NUM_STATES = CRASH + 1

//state singleton
var s = newState()

// if the requested state update is valid from the current state, moves the
// next state and updates any go routines waiting on the state update.
// returns a boolean if the update cannot be done and an error explaining
// why
func Update(nextState State)(bool,error){
	return s.update(nextState)
}

// gets the current state under a read lock
func Get()State{
	return s.get()
}

// waits to be notified and then returns an update
func GetUpdate()State {
	return s.getUpdate()
}

// if the the passed state is the next state update, waits until that update
// happens. return true if the waited state is the current state
func WaitOn(expected State, timeout time.Duration)(bool, error){
	return s.waitOn(expected, timeout)
}


