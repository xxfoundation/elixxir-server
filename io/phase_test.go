////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/services"
	"testing"
)

// Test that post phase properly sends the results to the phase via mockPhase
func TestPostPhase(t *testing.T) {

	numSlots := 3

	//Get a mock phase
	mockPhase := &MockPhase{}

	//Build a mock mockBatch to receive
	mockBatch := mixmessages.Batch{}

	for i := 0; i < numSlots; i++ {
		mockBatch.Slots = append(mockBatch.Slots,
			&mixmessages.Slot{
				PayloadA: []byte{byte(i)},
			})
	}

	//receive the mockBatch
	err := PostPhase(mockPhase, &mockBatch, nil)

	if err != nil {
		t.Errorf("PostPhase: Unexpected error returned: %+v", err)
	}

	for index := range mockBatch.Slots {
		if mockPhase.chunks[index].Begin() != uint32(index) {
			t.Errorf("PostPhase: output chunk not equal to passed;"+
				"Expected: %v, Recieved: %v", index, mockPhase.chunks[index].Begin())
		}

		if mockPhase.indices[index] != uint32(index) {
			t.Errorf("PostPhase: output index  not equal to passed;"+
				"Expected: %v, Recieved: %v", index, mockPhase.indices[index])
		}
	}

	mockBatch.Slots[0].Salt = []byte{42}
	mockBatch.Round = &mixmessages.RoundInfo{}

	err = PostPhase(mockPhase, &mockBatch, nil)

	if err == nil {
		t.Errorf("PostPhase: did not error when expected")
	}
}

var receivedBatch *mixmessages.Batch

// Tests that a batch sent via transmit phase arrives correctly
func TestTransmitPhase(t *testing.T) {

	// Setup the network
	comms, topology := buildTestNetworkComponents(
		[]*node.Implementation{nil, mockPostPhaseImplementation()}, 10)
	defer Shutdown(comms)

	// Build the mock functions called by the transmitter
	chunkCnt := uint32(0)
	batchSize := uint32(5)
	roundID := id.Round(5)
	phaseTy := phase.Type(2)

	getChunk := func() (services.Chunk, bool) {
		if chunkCnt < batchSize {
			chunk := services.NewChunk(chunkCnt, chunkCnt+1)
			chunkCnt++
			return chunk, true
		}
		return services.NewChunk(0, 0), false
	}

	getMsg := func(index uint32) *mixmessages.Slot {
		return &mixmessages.Slot{PayloadA: []byte{0}}
	}

	m := func(tag string) {}

	//call the transmitter
	err := TransmitPhase(comms[0], batchSize, roundID, phaseTy, getChunk,
		getMsg, topology, topology.GetNodeAtIndex(0), m)

	if err != nil {
		t.Errorf("TransmitPhase: Unexpected error: %+v", err)
	}

	//Check that what was receivedFinishRealtime is correct
	if id.Round(receivedBatch.Round.ID) != roundID {
		t.Errorf("TransmitPhase: Incorrect round ID"+
			"Expected: %v, Recieved: %v", roundID, receivedBatch.Round.ID)
	}

	if phase.Type(receivedBatch.FromPhase) != phaseTy {
		t.Errorf("TransmitPhase: Incorrect Phase type"+
			"Expected: %v, Recieved: %v", phaseTy, receivedBatch.FromPhase)
	}

	if uint32(len(receivedBatch.Slots)) != batchSize {
		t.Errorf("TransmitPhase: Recieved Batch of wrong size"+
			"Expected: %v, Recieved: %v", batchSize,
			uint32(len(receivedBatch.Slots)))
	}
}

func mockPostPhaseImplementation() *node.Implementation {
	impl := node.NewImplementation()
	impl.Functions.PostPhase = func(batch *mixmessages.Batch, auth *connect.Auth) error {
		receivedBatch = batch
		return nil
	}
	return impl
}
