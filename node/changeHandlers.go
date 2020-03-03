package node

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/server/server/state"
	"time"
)


func NotStarted(m state.Machine)  {
	// all the server startup code

	go generalUpdate(current.NOT_STARTED, current.WAITING, m)
	return
}

func Waiting(m state.Machine)  {
	go generalUpdate(current.WAITING, current.PRECOMPUTING, m)
	return
}

func Precomputing(m state.Machine)  {
	// start pre-precomputation

	go generalUpdate(current.PRECOMPUTING, current.STANDBY, m)
	return
}

func Standby(m state.Machine)  {
	go generalUpdate(current.STANDBY, current.REALTIME, m)


}

func Realtime(m state.Machine)  {
	go generalUpdate(current.REALTIME, current.COMPLETED, m)
}

func Completed(m state.Machine)  {
	go generalUpdate(current.ERROR, current.WAITING, m)
}

func NewStateChanges() {
	//return state changes arr
	//create the state change function table
	var stateChanges [current.NUM_STATES]state.Change

	stateChanges[current.NOT_STARTED]
}