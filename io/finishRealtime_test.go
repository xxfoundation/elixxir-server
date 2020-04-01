////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/services"
	"testing"
	"time"
)

var receivedFinishRealtime = make(chan *mixmessages.RoundInfo, 100)
var getMessage = func(index uint32) *mixmessages.Slot {
	return &mixmessages.Slot{}
}

func MockFinishRealtimeImplementation() *node.Implementation {
	impl := node.NewImplementation()
	impl.Functions.FinishRealtime = func(message *mixmessages.RoundInfo, auth *connect.Auth) error {
		receivedFinishRealtime <- message
		return nil
	}
	return impl
}

// Test that TransmitFinishRealtime correctly broadcasts message
// to all other nodes
func TestSendFinishRealtime(t *testing.T) {
	instance, _, _, _, _, _ := setup(t)

	//Setup the network
	numNodes := 4
	numRecv := 0
	comms, topology := buildTestNetworkComponents(
		[]*node.Implementation{
			MockFinishRealtimeImplementation(),
			MockFinishRealtimeImplementation(),
			MockFinishRealtimeImplementation(),
			MockFinishRealtimeImplementation(),
			MockFinishRealtimeImplementation()}, 0)
	defer Shutdown(comms)

	const numSlots = 10
	const numChunks = numSlots / 2

	rndID := id.Round(42)

	instance.GetRoundManager().AddRound(round.NewDummyRound(rndID, numSlots, t))

	chunkInputChan := make(chan services.Chunk, numChunks)

	getChunk := func() (services.Chunk, bool) {
		chunk, ok := <-chunkInputChan
		return chunk, ok
	}
	errCH := make(chan error)

	var err error
	go func() {
		err = TransmitFinishRealtime(rndID, instance, getChunk)
		errCH <- err
	}()

	for i := 0; i < numChunks; i++ {
		chunkInputChan <- services.NewChunk(uint32(i*2), uint32(i*2+1))
	}

	close(chunkInputChan)

	var cr *round.CompletedRound

	for cr == nil {
		cr, _ = instance.GetCompletedBatchQueue().Receive()
		time.Sleep(1 * time.Millisecond)
	}

	if len(cr.Round) != numSlots {
		t.Errorf("TransmitFinishRealtime: did not recieve the correct: "+
			"number of chunks; expected: %v, recieved: %v", numSlots,
			len(cr.Round))
	}

	goErr := <-errCH

	if goErr != nil {
		t.Errorf("TransmitFinishRealtime: Unexpected error: %+v", err)
	}

Loop:
	for {
		select {
		case msg := <-receivedFinishRealtime:
			if id.Round(msg.ID) != rndID {
				t.Errorf("TransmitFinishRealtime: Incorrect round ID"+
					"Expected: %v, Received: %v", rndID, msg.ID)
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

func MockFinishRealtimeImplementation_Error() *node.Implementation {
	impl := node.NewImplementation()
	impl.Functions.FinishRealtime = func(message *mixmessages.RoundInfo, auth *connect.Auth) error {
		return errors.New("Test error")
	}
	return impl
}

func TestTransmitFinishRealtime_Error(t *testing.T) {
	instance, _, _, _, _, _ := setup(t)

	//Setup the network
	comms, topology := buildTestNetworkComponents(
		[]*node.Implementation{
			MockFinishRealtimeImplementation_Error(),
			MockFinishRealtimeImplementation_Error(),
			MockFinishRealtimeImplementation_Error(),
			MockFinishRealtimeImplementation_Error(),
			MockFinishRealtimeImplementation_Error()}, 0)
	defer Shutdown(comms)

	rndID := id.Round(42)

	const numSlots = 10
	const numChunks = numSlots / 2

	instance.GetRoundManager().AddRound(round.NewDummyRound(rndID, numSlots, t))

	chunkInputChan := make(chan services.Chunk, numChunks)

	getChunk := func() (services.Chunk, bool) {
		chunk, ok := <-chunkInputChan
		return chunk, ok
	}
	errCH := make(chan error)

	go func() {
		err := TransmitFinishRealtime(rndID, instance, getChunk)
		errCH <- err
	}()

	go func() {
		for i := 0; i < numChunks; i++ {
			chunkInputChan <- services.NewChunk(uint32(i*2), uint32(i*2+1))
		}

		close(chunkInputChan)
	}()

	var cr *round.CompletedRound

	for cr == nil {
		cr, _ = instance.GetCompletedBatchQueue().Receive()
		time.Sleep(1 * time.Millisecond)
	}

	if len(cr.Round) != numSlots {
		t.Errorf("TransmitFinishRealtime: did not recieve the correct: "+
			"number of chunks; expected: %v, recieved: %v", numSlots,
			len(cr.Round))
	}

	goErr := <-errCH

	if goErr == nil {
		t.Error("SendFinishRealtime: error did not occur when provoked")
	}
}
