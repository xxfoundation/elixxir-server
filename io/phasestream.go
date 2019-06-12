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

	header := mixmessages.BatchInfo{
		Round: &mixmessages.RoundInfo{
			ID: uint64(roundID),
		},
		ForPhase: int32(phaseTy),
	}

	streamClient, cancel, err := network.GetPostPhaseStreamClient(recipient, header)
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
	slot, err := stream.Recv()
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
	}

	if err != io.EOF {
		return err
	}

	// Send ack back to client
	ack := mixmessages.Ack{
		Error: "",
	}

	return stream.SendAndClose(&ack)
}
