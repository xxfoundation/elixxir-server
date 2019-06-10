////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"context"
	"errors"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/services"
	"google.golang.org/grpc/metadata"
	"io"
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
				MessagePayload: []byte{byte(i)},
			})
	}

	//receive the mockBatch
	err := PostPhase(mockPhase, &mockBatch)

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

	err = PostPhase(mockPhase, &mockBatch)

	if err == nil {
		t.Errorf("PostPhase: did not error when expected")
	}
}

var mockStreamSlotIndex int

type MockStreamPostPhaseServer struct {
	batch mixmessages.Batch
}

func (stream MockStreamPostPhaseServer) SendAndClose(*mixmessages.Ack) error {
	if len(stream.batch.Slots) == mockStreamSlotIndex {
		return nil
	}
	return errors.New("stream closed without all slots being received")
}

func (stream MockStreamPostPhaseServer) Recv() (*mixmessages.Slot, error) {
	if mockStreamSlotIndex >= len(stream.batch.Slots) {
		return nil, io.EOF
	}
	slot := stream.batch.Slots[mockStreamSlotIndex]
	mockStreamSlotIndex++
	return slot, nil
}

func (MockStreamPostPhaseServer) SetHeader(metadata.MD) error {
	return nil
}

func (MockStreamPostPhaseServer) SendHeader(metadata.MD) error {
	return nil
}

func (MockStreamPostPhaseServer) SetTrailer(metadata.MD) {
}

func (MockStreamPostPhaseServer) Context() context.Context {
	return nil
}

func (MockStreamPostPhaseServer) SendMsg(m interface{}) error {
	return nil
}

func (MockStreamPostPhaseServer) RecvMsg(m interface{}) error {
	return nil
}

// Test that post phase properly sends the results to the phase via mockPhase
func TestStreamPostPhase(t *testing.T) {

	numSlots := 3

	//Get a mock phase
	mockPhase := &MockPhase{}

	//Build a mock mockBatch to receive
	mockBatch := mixmessages.Batch{}
	for i := 0; i < numSlots; i++ {
		mockBatch.Slots = append(mockBatch.Slots,
			&mixmessages.Slot{
				MessagePayload: []byte{byte(i)},
			})
	}

	// receive the mockBatch into the mock stream 'buffer'
	mockStreamServer := MockStreamPostPhaseServer{batch: mockBatch}

	err := StreamPostPhase(mockPhase, mockStreamServer)

	if err != nil {
		t.Errorf("StreamPostPhase: Unexpected error returned: %+v", err)
	}

	for index := range mockBatch.Slots {
		if mockPhase.chunks[index].Begin() != uint32(index) {
			t.Errorf("StreamPostPhase: output chunk not equal to passed;"+
				"Expected: %v, Recieved: %v", index, mockPhase.chunks[index].Begin())
		}

		if mockPhase.indices[index] != uint32(index) {
			t.Errorf("StreamPostPhase: output index  not equal to passed;"+
				"Expected: %v, Recieved: %v", index, mockPhase.indices[index])
		}
	}
}

var receivedBatch *mixmessages.Batch

//var receivedStreamServer mixmessages.Node_StreamPostPhaseServer

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
			chunk, ok := services.NewChunk(chunkCnt, chunkCnt+1), false
			chunkCnt++
			return chunk, ok
		}
		return services.NewChunk(0, 0), true
	}

	getMsg := func(index uint32) *mixmessages.Slot {
		return &mixmessages.Slot{MessagePayload: []byte{0}}
	}

	//call the transmitter
	err := TransmitPhase(comms[0], batchSize, roundID, phaseTy, getChunk,
		getMsg, topology, topology.GetNodeAtIndex(0))

	if err != nil {
		t.Errorf("TransmitPhase: Unexpected error: %+v", err)
	}

	//Check that what was receivedFinishRealtime is correct
	if id.Round(receivedBatch.Round.ID) != roundID {
		t.Errorf("TransmitPhase: Incorrect round ID"+
			"Expected: %v, Recieved: %v", roundID, receivedBatch.Round.ID)
	}

	if phase.Type(receivedBatch.ForPhase) != phaseTy {
		t.Errorf("TransmitPhase: Incorrect Phase type"+
			"Expected: %v, Recieved: %v", phaseTy, receivedBatch.ForPhase)
	}

	if uint32(len(receivedBatch.Slots)) != batchSize {
		t.Errorf("TransmitPhase: Recieved Batch of wrong size"+
			"Expected: %v, Recieved: %v", batchSize,
			uint32(len(receivedBatch.Slots)))
	}
}

func mockPostPhaseImplementation() *node.Implementation {
	impl := node.NewImplementation()
	impl.Functions.PostPhase = func(batch *mixmessages.Batch) {
		receivedBatch = batch
	}
	return impl
}

//
//func mockStreamPostPhaseImplementation() *node.Implementation {
//	impl := node.NewImplementation()
//	impl.Functions.StreamPostPhase = func(stream mixmessages.Node_StreamPostPhaseServer) error {
//
//		receivedStreamServer = stream
//		return nil
//	}
//}
