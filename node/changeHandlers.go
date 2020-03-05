////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package node

import (
	"github.com/pkg/errors"
	"github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/server/state"
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

func Precomputing(instance *server.Instance, newRoundTimeout int) (state.Change, error) {
	// Add round.queue to instance, get that here and use it to get new round
	// start pre-precomputation
	roundInfo := <-instance.GetCreateRoundQueue()
	roundID := id.Round(roundInfo.ID)
	topology := roundInfo.GetTopology()
	nodeIDs := make([]*id.Node, 0)
	for _, s := range topology {
		nodeIDs = append(nodeIDs, id.NewNodeFromBytes([]byte(s)))
	}
	circuit := connect.NewCircuit(nodeIDs)

	//Build the components of the round
	phases, phaseResponses := NewRoundComponents(
		instance.GetGraphGenerator(),
		circuit,
		instance.GetID(),
		instance,
		instance.GetBatchSize(),
		newRoundTimeout)

	//Build the round
	rnd := round.New(
		instance.GetGroup(),
		instance.GetUserRegistry(),
		roundID, phases, phaseResponses,
		circuit,
		instance.GetID(),
		instance.GetBatchSize(),
		instance.GetRngStreamGen(),
		instance.GetIP())

	//Add the round to the manager
	instance.GetRoundManager().AddRound(rnd)

	jwalterweatherman.INFO.Printf("[%s]: RID %d CreateNewRound COMPLETE", instance,
		roundID)

	if circuit.IsFirstNode(instance.GetID()) {
		err := StartLocalPrecomp(instance, roundID, roundInfo.BatchSize)
		if err != nil {
			return nil, errors.WithMessage(err, "Failed to TransmitCreateNewRound")
		}
	}

	return nil, nil
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
