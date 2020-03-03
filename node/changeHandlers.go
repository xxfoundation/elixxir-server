////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package node

import (
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/server/server/state"
)


func NotStarted(from current.Activity) error  {
	// all the server startup code

	return nil
}

func Waiting(from current.Activity) error  {
	// start waiting process
	return nil
}

func Precomputing(from current.Activity) error {
	// start pre-precomputation

	return nil
}

func Standby(from current.Activity) error {
	// start standby process
	return nil

}

func Realtime(from current.Activity) error  {
	// start realtime
	return nil

}

func Completed(from current.Activity) error {
	// start completed
	return nil
}


func NewStateChanges() [current.NUM_STATES]state.Change {
	//return state changes arr
	//create the state change function table
	var stateChanges [current.NUM_STATES]state.Change

	stateChanges[current.NOT_STARTED] = NotStarted
	stateChanges[current.WAITING] = Waiting
	stateChanges[current.PRECOMPUTING] = Precomputing
	stateChanges[current.STANDBY] = Standby
	stateChanges[current.REALTIME] = Realtime
	stateChanges[current.COMPLETED] = Completed


	return stateChanges
}