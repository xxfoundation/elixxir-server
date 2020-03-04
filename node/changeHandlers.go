////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package node

import (
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/io"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/server/state"
	"time"
)

func Dummy(from current.Activity) error {
	return nil
}

func NotStarted(from current.Activity) error {
	// all the server startup code

	return nil
}

func Waiting(from current.Activity) error {
	// start waiting process
	return nil
}

func Precomputing(instance server.Instance, newRoundTimeout int) state.Change {
	// Add round.queue to instance, get that here and use it to get new round
	// start pre-precomputation
	jww.INFO.Printf("[%s]: RID %d CreateNewRound RECIEVE", instance,
		roundID)

	//Build the components of the round
	phases, phaseResponses := NewRoundComponents(
		instance.GetGraphGenerator(),
		instance.GetTopology(),
		instance.GetID(),
		instance,
		instance.GetBatchSize(),
		newRoundTimeout)

	//Build the round
	rnd := round.New(
		instance.GetGroup(),
		instance.GetUserRegistry(),
		roundID, phases, phaseResponses,
		instance.GetTopology(),
		instance.GetID(),
		instance.GetBatchSize(),
		instance.GetRngStreamGen(),
		instance.GetIP())

	//Add the round to the manager
	instance.GetRoundManager().AddRound(rnd)

	jww.INFO.Printf("[%s]: RID %d CreateNewRound COMPLETE", instance,
		roundID)

	instance.RunFirstNode(instance, roundBufferTimeout*time.Second,
		io.TransmitCreateNewRound, node.MakeStarter(params.Batch))

	return nil
}

func Standby(from current.Activity) error {
	// start standby process
	return nil

}

func Realtime(from current.Activity) error {
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

	stateChanges[current.NOT_STARTED] = Dummy
	stateChanges[current.WAITING] = Dummy
	stateChanges[current.PRECOMPUTING] = Dummy
	stateChanges[current.STANDBY] = Dummy
	stateChanges[current.REALTIME] = Dummy
	stateChanges[current.COMPLETED] = Dummy

	return stateChanges
}
