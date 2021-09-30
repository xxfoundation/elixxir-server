///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/testUtil"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/primitives/id"
	"testing"
	"time"
)

func TestTransmitPrecompTestBatch(t *testing.T) {
	instance, _, _, _, _, _, _, _ := setup(t)

	//Setup the network
	numNodes := 4
	numRecv := 0
	comms, topology := buildTestNetworkComponents(
		[]*node.Implementation{
			MockPrecompTestBatchImplementation(),
			MockPrecompTestBatchImplementation(),
			MockPrecompTestBatchImplementation(),
			MockPrecompTestBatchImplementation(),
			MockPrecompTestBatchImplementation()}, 0, t)
	defer Shutdown(comms)

	const numSlots = 10

	const numChunks = numSlots / 2

	response := phase.NewResponse(phase.ResponseDefinition{
		PhaseAtSource:  phase.PrecompDecrypt,
		ExpectedStates: []phase.State{phase.Active},
		PhaseToExecute: phase.PrecompDecrypt})

	roundID := id.Round(0)
	grp := initImplGroup()
	p := testUtil.InitMockPhase(t)
	p.Ptype = phase.PrecompDecrypt
	responseMap := make(phase.ResponseMap)
	responseMap["RealPermuteVerification"] = response

	rnd, err := round.New(grp, nil, roundID, []phase.Phase{p}, responseMap, topology, topology.GetNodeAtIndex(0), numSlots, instance.GetRngStreamGen(), nil, "0.0.0.0", nil, nil, nil)
	if err != nil {
		t.Error()
	}
	instance.GetRoundManager().AddRound(rnd)

	errCH := make(chan error)
	go func() {
		err = TransmitPrecompTestBatch(roundID, instance)
		errCH <- err
	}()

	goErr := <-errCH

	if goErr != nil {
		t.Errorf("TransmitFinishRealtime: Unexpected error: %+v", err)
	}

Loop:
	for {
		select {
		case msg := <-receivedPrecompTestBatch:
			if id.Round(msg.ID) != roundID {
				t.Errorf("TransmitPrecompTestBatch: Incorrect round ID"+
					"Expected: %v, Received: %v", roundID, msg.ID)
			}
			numRecv++
			if numRecv == numNodes {
				break Loop
			}
		case <-time.After(5 * time.Second):
			t.Errorf("Test timed out!")
			break Loop
		}
	}

}

var receivedPrecompTestBatch = make(chan *mixmessages.RoundInfo, 100)

func MockPrecompTestBatchImplementation() *node.Implementation {
	impl := node.NewImplementation()

	impl.Functions.PrecompTestBatch = func(streamServer mixmessages.Node_PrecompTestBatchServer,
		message *mixmessages.RoundInfo, auth *connect.Auth) error {
		receivedPrecompTestBatch <- message
		streamServer.SendAndClose(&messages.Ack{})
		return nil
	}
	return impl
}
