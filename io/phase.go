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
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/services"
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
		FromPhase: int32(phaseTy),
		Slots:     make([]*mixmessages.Slot, batchSize),
	}

	// For each message chunk (slot), fill the slots buffer
	// Note that this will panic if there are more slots than batchSize
	// (shouldn't be possible?)
	for chunk, finish := getChunk(); finish; chunk, finish = getChunk() {
		for i := chunk.Begin(); i < chunk.End(); i++ {
			msg := getMessage(i)
			batch.Slots[i] = msg
		}
	}

	name := services.NameStringer("HostUnknown:PortUnknown",
		topology.GetNodeLocation(nodeID), topology.Len())
	jww.INFO.Printf("[%s]: RID %d TransmitPhase FOR \"%s\" COMPLETE/SEND",
		name, roundID, phaseTy)

	// Make sure the comm doesn't return an Ack with an error message
	ack, err := network.SendPostPhase(recipient, batch)
	if ack != nil && ack.Error != "" {
		err = errors.Errorf("Remote Server Error: %s", ack.Error)
	}
	return err
}

// PostPhase implements the server gRPC handler for posting a
// phase from another node
func PostPhase(p phase.Phase, batch *mixmessages.Batch) error {

	// Send a chunk per slot
	for index, message := range batch.Slots {
		curIdx := uint32(index)
		err := p.Input(curIdx, message)
		if err != nil {
			return errors.Errorf("Error on Round %v, phase \"%s\" "+
				"slot %d, contents: %v: %v", batch.Round.ID, phase.Type(batch.FromPhase),
				curIdx, message, err)
		}
		chunk := services.NewChunk(curIdx, curIdx+1)
		p.Send(chunk)
	}

	return nil
}
