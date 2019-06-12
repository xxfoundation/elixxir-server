////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
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

	// Create the message structure to send the messages
	batchInfo := mixmessages.BatchInfo{
		Round: &mixmessages.RoundInfo{
			ID: uint64(roundID),
		},
		ForPhase: int32(phaseTy),
	}

	// Get stream client context
	ctx, cancel := network.GetPostPhaseStreamContext(batchInfo)

	// Get stream client
	streamClient, err := network.GetPostPhaseStream(recipient, ctx)

	if err != nil {
		jww.ERROR.Printf("Error on comm, unable to get streaming client: %+v", err)
	}

	// For each message chunk (slot) stream it out
	for chunk, finish := getChunk(); !finish; chunk, finish = getChunk() {
		for i := chunk.Begin(); i < chunk.End(); i++ {
			msg := getMessage(i)

			err := streamClient.Send(msg)
			if err != nil {
				jww.ERROR.Printf("Error on comm, not able to send slot: %+v", err)
			}
		}
	}

	ack, err := streamClient.CloseAndRecv()

	if err != nil {
		return err
	}

	// Make sure the comm doesn't return an Ack with an error message
	if ack != nil && ack.Error != "" {
		err = errors.Errorf("Remote Server Error: %s", ack.Error)
		return err
	}

	cancel()

	return nil
}

// StreamPostPhase implements the server gRPC handler for posting a
// phase from another node
func StreamPostPhase(p phase.Phase, stream mixmessages.Node_StreamPostPhaseServer) error {

	// Send a chunk for each slot received until EOF
	// then send ack back to client.
	index := uint32(0)

	for {
		slot, err := stream.Recv()
		// If we are at end of receiving
		// send ack and finish
		if err == io.EOF {
			ack := mixmessages.Ack{
				Error: "",
			}

			err = stream.SendAndClose(&ack)

			return err
		}

		if err != nil {
			return err
		}

		err = p.Input(index, slot)
		if err != nil {
			return errors.Errorf("Error on slot %d: %v", index, err)
		}

		chunk := services.NewChunk(index, index+1)
		p.Send(chunk)

		index++
	}
}
