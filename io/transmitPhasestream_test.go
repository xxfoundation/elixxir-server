///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package io

import (
	"context"
	"errors"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/comms/testkeys"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/elixxir/server/testUtil"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/utils"
	"google.golang.org/grpc/metadata"
	"io"
	"testing"
)

var mockIndex int

type MockTransmitStream struct {
	batch mixmessages.Batch
}

func (stream MockTransmitStream) SendAndClose(*messages.Ack) error {
	if len(stream.batch.Slots) == mockIndex {
		return nil
	}
	return errors.New("stream closed without all slots being received")
}

func (stream MockTransmitStream) Recv() (*mixmessages.Slot, error) {
	if mockIndex >= len(stream.batch.Slots) {
		return nil, io.EOF
	}
	slot := stream.batch.Slots[mockIndex]
	mockIndex++
	return slot, nil
}

func (MockTransmitStream) SetHeader(metadata.MD) error {
	return nil
}

func (MockTransmitStream) SendHeader(metadata.MD) error {
	return nil
}

func (MockTransmitStream) SetTrailer(metadata.MD) {
}

func (stream MockTransmitStream) Context() context.Context {
	return nil
}

func (MockTransmitStream) SendMsg(m interface{}) error {
	return nil
}

func (MockTransmitStream) RecvMsg(m interface{}) error {
	return nil
}

// Test that post phase properly sends the results to the phase via mockPhase
func TestStreamPostPhase(t *testing.T) {

	batchSize := 1

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
	mockStreamServer := MockTransmitStream{batch: mockBatch}

	_, err := StreamPostPhase(mockPhase, uint32(batchSize), mockStreamServer)

	if err != nil {
		t.Errorf("StreamPostPhase: Unexpected error returned: %+v", err)
	}

	for index := range mockBatch.Slots {
		if mockPhase.chunks[index].Begin() != uint32(index) {
			t.Errorf("StreamPostPhase: output chunk not equal to passed;"+
				"Expected: %v, Received: %v", index, mockPhase.chunks[index].Begin())
		}

		if mockPhase.indices[index] != uint32(index) {
			t.Errorf("StreamPostPhase: output index  not equal to passed;"+
				"Expected: %v, Received: %v", index, mockPhase.indices[index])
		}
	}
}

// Tests that a batch sent via transmit phase arrives correctly
func TestStreamTransmitPhase(t *testing.T) {
	instance, nodeAddr := mockInstance(t, mockStreamPostPhaseImplementation)

	// Build the mock functions called by the transmitter
	chunkCnt := uint32(0)
	batchSize := uint32(5)
	roundID := id.Round(5)
	phaseTy := phase.Type(0)

	response := phase.NewResponse(phase.ResponseDefinition{
		PhaseAtSource:  phase.PrecompDecrypt,
		ExpectedStates: []phase.State{phase.Active},
		PhaseToExecute: phase.PrecompDecrypt})

	grp := initImplGroup()

	p := testUtil.InitMockPhase(t)
	p.Ptype = phase.PrecompDecrypt
	responseMap := make(phase.ResponseMap)
	responseMap["PrecompDecrypt"] = response

	topology := connect.NewCircuit([]*id.ID{instance.GetID()})

	cert, _ := utils.ReadFile(testkeys.GetNodeCertPath())
	nodeHost, _ := connect.NewHost(instance.GetID(), nodeAddr, cert, connect.GetDefaultHostParams())
	topology.AddHost(nodeHost)
	_, err := instance.GetNetwork().AddHost(instance.GetID(), nodeAddr, cert, connect.GetDefaultHostParams())
	if err != nil {
		t.Errorf("Failed to add host to instance: %v", err)
	}

	rnd, err := round.New(grp, nil, roundID, []phase.Phase{p}, responseMap, topology,
		topology.GetNodeAtIndex(0), batchSize, instance.GetRngStreamGen(), nil,
		"0.0.0.0", nil, nil)
	if err != nil {
		t.Error()
	}

	instance.GetRoundManager().AddRound(rnd)

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

	// call the transmitter
	err = StreamTransmitPhase(roundID, instance, getChunk, getMsg)

	if err != nil {
		t.Errorf("StreamTransmitPhase failed: %v", err)
		t.Fail()
	}

	//Check that what was received is correct
	if id.Round(receivedBatch.Round.ID) != roundID {
		t.Errorf("StreamTransmitPhase: Incorrect round ID"+
			"Expected: %v, Received: %v", roundID, receivedBatch.Round.ID)
	}

	if phase.Type(receivedBatch.FromPhase) != phaseTy {
		t.Errorf("StreamTransmitPhase: Incorrect Phase type"+
			"Expected: %v, Received: %v", phaseTy, receivedBatch.FromPhase)
	}

	if uint32(len(receivedBatch.Slots)) != batchSize {
		t.Errorf("StreamTransmitPhase: Received Batch of wrong size"+
			"Expected: %v, Received: %v", batchSize,
			uint32(len(receivedBatch.Slots)))
	}

}

func mockStreamPostPhaseImplementation(instance *internal.Instance) *node.Implementation {
	impl := node.NewImplementation()
	impl.Functions.StreamPostPhase = func(stream mixmessages.Node_StreamPostPhaseServer, auth *connect.Auth) error {
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
			ack := messages.Ack{
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
