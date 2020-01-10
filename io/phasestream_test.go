////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"context"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/services"
	"google.golang.org/grpc/metadata"
	"io"
	"testing"
)

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

func (stream MockStreamPostPhaseServer) Context() context.Context {
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

	batchSize := 3

	//Get a mock phase
	mockPhase := &MockPhase{}

	//Build a mock mockBatch to receive
	mockBatch := mixmessages.Batch{}
	for i := 0; i < batchSize; i++ {
		mockBatch.Slots = append(mockBatch.Slots,
			&mixmessages.Slot{
				Index:    uint32(i),
				PayloadA: []byte{byte(i)},
			})
	}

	// receive the mockBatch into the mock stream 'buffer'
	mockStreamServer := MockStreamPostPhaseServer{batch: mockBatch}

	err := StreamPostPhase(mockPhase, uint32(batchSize), mockStreamServer)

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

// Tests that a batch sent via transmit phase arrives correctly
func TestStreamTransmitPhase(t *testing.T) {

	// Setup the network
	comms, topology := buildTestNetworkComponents(
		[]*node.Implementation{nil, mockStreamPostPhaseImplementation()}, 10)
	defer Shutdown(comms)

	// Build the mock functions called by the transmitter
	chunkCnt := uint32(0)
	batchSize := uint32(5)
	roundID := id.Round(5)
	phaseTy := phase.Type(2)

	getChunk := func() (services.Chunk, bool) {
		if chunkCnt < batchSize {
			chunk, _ := services.NewChunk(chunkCnt, chunkCnt+1), true
			chunkCnt++
			return chunk, true
		}
		return services.NewChunk(0, 0), false
	}

	getMsg := func(index uint32) *mixmessages.Slot {
		return &mixmessages.Slot{
			Index:    index,
			PayloadA: []byte{0},
		}
	}

	m := func(tag string) {}

	// call the transmitter
	err := StreamTransmitPhase(comms[0], batchSize, roundID, phaseTy, getChunk,
		getMsg, topology, topology.GetNodeAtIndex(0), m)

	if err != nil {
		t.Errorf("StreamTransmitPhase failed %v", err)
	}

	//Check that what was received is correct
	if id.Round(receivedBatch.Round.ID) != roundID {
		t.Errorf("StreamTransmitPhase: Incorrect round ID"+
			"Expected: %v, Recieved: %v", roundID, receivedBatch.Round.ID)
	}

	if phase.Type(receivedBatch.FromPhase) != phaseTy {
		t.Errorf("StreamTransmitPhase: Incorrect Phase type"+
			"Expected: %v, Recieved: %v", phaseTy, receivedBatch.FromPhase)
	}

	if uint32(len(receivedBatch.Slots)) != batchSize {
		t.Errorf("StreamTransmitPhase: Recieved Batch of wrong size"+
			"Expected: %v, Recieved: %v", batchSize,
			uint32(len(receivedBatch.Slots)))
	}

}

func mockStreamPostPhaseImplementation() *node.Implementation {
	impl := node.NewImplementation()
	impl.Functions.StreamPostPhase = func(stream mixmessages.
		Node_StreamPostPhaseServer, auth *connect.Auth) error {
		receivedBatch = &mixmessages.Batch{}
		return mockStreamPostPhase(stream)
	}

	return impl
}

func mockStreamPostPhase(stream mixmessages.Node_StreamPostPhaseServer) error {

	// Receive all slots and on EOF store all data
	// into a global received batch variable then
	// send ack back to client.
	var slots []*mixmessages.Slot
	index := uint32(0)
	for {
		slot, err := stream.Recv()
		// If we are at end of receiving
		// send ack and finish
		if err == io.EOF {
			ack := mixmessages.Ack{
				Error: "",
			}

			batchInfo, err := node.GetPostPhaseStreamHeader(stream)
			if err != nil {
				return err
			}

			// Create batch using batch info header
			// and temporary slot buffer contents
			receivedBatch = &mixmessages.Batch{
				Round:     batchInfo.Round,
				FromPhase: batchInfo.FromPhase,
				Slots:     slots,
			}

			err = stream.SendAndClose(&ack)
			return err
		}

		// If we have another error, return err
		if err != nil {
			return err
		}

		// Store slot received into temporary buffer
		slots = append(slots, slot)

		index++
	}

}
