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
	"gitlab.com/elixxir/comms/mixmessages"
	comm "gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/services"
)

// TransmitPhase sends a cMix Batch of messages to the provided Node.
func TransmitPhase(batchSize uint32, roundID id.Round, phaseTy phase.Type,
	getChunk phase.GetChunk, getMessage phase.GetMessage, nal *services.NodeAddressList) error {

	recipient := nal.GetNextNodeAddress()

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
	ack, err := comm.SendPostPhase(recipient.Address, recipient.Cert, batch)
	if ack != nil && ack.Error != "" {
		err = errors.Errorf("Remote Server Error: %s", ack.Error)
	}
	return err
}

// PostPhase implements the server gRPC handler for posting a
// phase from another node
func PostPhase(p *phase.Phase, batch *mixmessages.Batch) error {

	// Send a chunk per slot
	graph := p.GetGraph()
	stream := graph.GetStream()
	for index, messages := range batch.Slots {
		curIdx := uint32(index)
		err := stream.Input(curIdx, messages)
		if err != nil {
			return errors.Errorf("Error on slot %d: %v", curIdx,
				err)
		}
		chunk := services.NewChunk(curIdx, curIdx+1)
		graph.Send(chunk)
	}

	return nil
}
