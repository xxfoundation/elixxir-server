////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package io phase.go handles the endpoints and helper functions for
// receiving and sending batches of cMix messages through phases.

package io

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/services"
	"google.golang.org/grpc/metadata"
	"io"
)

// TransmitPhase sends a cMix Batch of messages to the provided Node.
func TransmitPhase(network *node.NodeComms, batchSize uint32,
	roundID id.Round, phaseTy phase.Type,
	getChunk phase.GetChunk, getMessage phase.GetMessage,
	topology *circuit.Circuit, nodeID *id.Node) error {

	recipient := topology.GetNextNode(nodeID)

	// Create the message structure to send the messages
	batch := &mixmessages.Batch{
		Round: &mixmessages.RoundInfo{
			ID: uint64(roundID),
		},
		ForPhase: int32(phaseTy),
		Slots:    make([]*mixmessages.Slot, batchSize),
	}

	// For each message chunk (slot), fill the slots buffer
	// Note that this will panic if there are more slots than batchSize
	// (shouldn't be possible?)
	for chunk, finish := getChunk(); !finish; chunk, finish = getChunk() {
		for i := chunk.Begin(); i < chunk.End(); i++ {
			msg := getMessage(i)
			batch.Slots[i] = msg
		}
	}

	// Make sure the comm doesn't return an Ack with an error message
	ack, err := network.SendPostPhase(recipient, batch)
	if ack != nil && ack.Error != "" {
		err = errors.Errorf("Remote Server Error: %s", ack.Error)
	}
	return err
}

// StreamTransmitPhase streams slot messages to the provided Node.
func StreamTransmitPhase(network *node.NodeComms, batchSize uint32,
	roundID id.Round, phaseTy phase.Type,
	getChunk phase.GetChunk, getMessage phase.GetMessage,
	topology *circuit.Circuit, nodeID *id.Node) error {

	recipient := topology.GetNextNode(nodeID)

	// Create the message structure to send the messages
	batchInfo := &mixmessages.BatchInfo{
		Round: &mixmessages.RoundInfo{
			ID: uint64(roundID),
		},
		ForPhase: int32(phaseTy),
	}

	// Create streaming context so you can close stream later
	ctx, cancel := connect.StreamingContext()

	// Create a new context with some metadata
	ctx = metadata.AppendToOutgoingContext(ctx,
		"BatchInfo", batchInfo.String())

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
	// Make sure the comm doesn't return an Ack with an error message
	if ack != nil && ack.Error != "" {
		err = errors.Errorf("Remote Server Error: %s", ack.Error)
	}

	cancel()

	return err
}

// PostPhase implements the server gRPC handler for posting a
// phase from another node
func PostPhase(p phase.Phase, batch *mixmessages.Batch) error {

	// Send a chunk per slot
	for index, messages := range batch.Slots {
		curIdx := uint32(index)
		err := p.Input(curIdx, messages)
		if err != nil {
			return errors.Errorf("Error on slot %d: %v", curIdx,
				err)
		}
		chunk := services.NewChunk(curIdx, curIdx+1)
		p.Send(chunk)
	}

	return nil
}

// StreamPostPhase implements the server gRPC handler for posting a
// phase from another node
func StreamPostPhase(p phase.Phase, server mixmessages.Node_StreamPostPhaseServer) error {

	// Send a chunk per received slot until EOF
	index := uint32(0)
	for {
		slot, err := server.Recv()
		index++
		if err == io.EOF {
			// TODO: send ack here

			return nil
		}
		if err != nil {
			// TODO: Log error
			return err
		}

		err = p.Input(index, slot)
		if err != nil {
			return errors.Errorf("Error on slot %d: %v", index,
				err)
		}

		chunk := services.NewChunk(index, index+1)
		p.Send(chunk)

	}
}
