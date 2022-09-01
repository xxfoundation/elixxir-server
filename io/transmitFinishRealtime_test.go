////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/elixxir/server/testUtil"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/primitives/id"
	"testing"
	"time"
)

var receivedFinishRealtime = make(chan *mixmessages.RoundInfo, 100)
var getMessage = func(index uint32) *mixmessages.Slot {
	return &mixmessages.Slot{
		Index: 24,
	}
}

func MockFinishRealtimeImplementation() *node.Implementation {
	impl := node.NewImplementation()
	impl.Functions.FinishRealtime = func(message *mixmessages.RoundInfo,
		streamServer mixmessages.Node_FinishRealtimeServer, auth *connect.Auth) error {
		receivedFinishRealtime <- message
		streamServer.SendAndClose(&messages.Ack{})
		return nil
	}
	return impl
}

// Test that TransmitFinishRealtime correctly broadcasts message
// to all other nodes
func TestSendFinishRealtime(t *testing.T) {
	instance, _, _, _, _, _, _, _ := setup(t)

	//Setup the network
	numNodes := 4
	numRecv := 0
	comms, topology := buildTestNetworkComponents(
		[]*node.Implementation{
			MockFinishRealtimeImplementation(),
			MockFinishRealtimeImplementation(),
			MockFinishRealtimeImplementation(),
			MockFinishRealtimeImplementation(),
			MockFinishRealtimeImplementation()}, 0, t)
	defer Shutdown(comms)

	const numSlots = 10

	const numChunks = numSlots / 2

	response := phase.NewResponse(phase.ResponseDefinition{
		PhaseAtSource:  phase.RealPermute,
		ExpectedStates: []phase.State{phase.Active},
		PhaseToExecute: phase.RealPermute})

	roundID := id.Round(0)
	grp := initImplGroup()
	p := testUtil.InitMockPhase(t)
	p.Ptype = phase.RealPermute
	responseMap := make(phase.ResponseMap)
	responseMap["RealPermuteVerification"] = response

	rnd, err := round.New(grp, roundID, []phase.Phase{p}, responseMap, topology, topology.GetNodeAtIndex(0), numSlots, instance.GetRngStreamGen(), nil, "0.0.0.0", nil, nil, nil, nil)
	if err != nil {
		t.Error()
	}
	instance.GetRoundManager().AddRound(rnd)

	chunkInputChan := make(chan services.Chunk, numChunks)

	getChunk := func() (services.Chunk, bool) {
		chunk, ok := <-chunkInputChan
		return chunk, ok
	}
	errCH := make(chan error)

	go func() {
		err = TransmitFinishRealtime(roundID, instance, getChunk, getMessage)
		errCH <- err
	}()

	for i := 0; i < numSlots; i++ {
		chunkInputChan <- services.NewChunk(uint32(i), uint32(i+1))
	}

	close(chunkInputChan)

	goErr := <-errCH

	if goErr != nil {
		t.Errorf("TransmitFinishRealtime: Unexpected error: %+v", err)
	}

Loop:
	for {
		select {
		case msg := <-receivedFinishRealtime:
			if id.Round(msg.ID) != roundID {
				t.Errorf("TransmitFinishRealtime: Incorrect round ID"+
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
