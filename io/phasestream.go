////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/services"
	"io"
)

// StreamTransmitPhase streams slot messages to the provided Node.
func StreamTransmitPhase(network *node.NodeComms, batchSize uint32,
	roundID id.Round, phaseTy phase.Type,
	getChunk phase.GetChunk, getMessage phase.GetMessage,
	topology *circuit.Circuit, nodeID *id.Node) error {

	recipient := topology.GetNextNode(nodeID)

	header := mixmessages.BatchInfo{
		Round: &mixmessages.RoundInfo{
			ID: uint64(roundID),
		},
		ForPhase: int32(phaseTy),
	}

	// This gets the streaming client which used to send slots
	// using the recipient node id and the batch info header
	// It's context must be canceled after receiving an ack
	streamClient, cancel, err := network.GetPostPhaseStreamClient(recipient, header)
	if err != nil {
		jww.ERROR.Printf("Error on comm, unable to get streaming client: %+v", err)
	}

	// For each message chunk (slot) stream it out
	for chunk, finish := getChunk(); !finish; chunk, finish = getChunk() {

		for i := chunk.Begin(); i < chunk.End(); i++ {
			fmt.Println("sending external: ", i)
			msg := getMessage(i)

			err := streamClient.Send(msg)
			if err != nil {
				jww.ERROR.Printf("Error on comm, not able to send slot: %+v", err)
			}
		}
	}

	// Receive ack and cancel client streaming context
	ack, err := streamClient.CloseAndRecv()

	jww.INFO.Printf("RID %d StreamTransmitPhase FOR \"%s\" COMPLETE/SEND",
		roundID, phaseTy)

	cancel()

	if err != nil {
		return err
	}

	// Make sure the comm doesn't return an Ack with an error message
	if ack != nil && ack.Error != "" {
		err = errors.Errorf("Remote Server Error: %s", ack.Error)
		return err
	}

	return nil
}

// StreamPostPhase implements the server gRPC handler for posting a
// phase from another node
func StreamPostPhase(p phase.Phase, batchSize uint32,
	stream mixmessages.Node_StreamPostPhaseServer) error {

	// Send a chunk for each slot received along with
	// its index until an error is received
	slot, err := stream.Recv()
	slotsReceived := uint32(0)
	for ; err == nil; slot, err = stream.Recv() {

		index := slot.Index

		phaseErr := p.Input(index, slot)
		if phaseErr != nil {
			jww.ERROR.Printf("Failed on phase input %v for slot %v ",
				index, slot)
			return phaseErr
		}

		chunk := services.NewChunk(index, index+1)
		p.Send(chunk)

		slotsReceived++
	}

	// Set error in ack message if we didn't receive all slots
	ack := mixmessages.Ack{
		Error: "",
	}
	if err != io.EOF {
		ack = mixmessages.Ack{
			Error: "failed to receive all slots: " + err.Error(),
		}
	}

	if slotsReceived != batchSize {
		jww.FATAL.Panicf(fmt.Sprintf("mismatch between batch size %v"+
			"and received num slots %v", batchSize, slotsReceived))
	}

	// Close the stream by sending ack
	// and returning whether it succeeded
	return stream.SendAndClose(&ack)
}
