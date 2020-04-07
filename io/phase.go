////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package io phase.go handles the endpoints and helper functions for
// receiving and sending batches of cMix messages through phases.

package io

import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/services"
	"strings"
)

// TransmitPhase sends a cMix Batch of messages to the provided Node.
func TransmitPhase(roundID id.Round, serverInstance phase.GenericInstance, getChunk phase.GetChunk,
	getMessage phase.GetMessage) error {

	instance, ok := serverInstance.(*server.Instance)
	if !ok {
		return errors.Errorf("Invalid server instance passed in")
	}

	//get the round so you can get its batch size
	r, err := instance.GetRoundManager().GetRound(roundID)
	if err != nil {
		return errors.Errorf("Received completed batch for round %v that doesn't exist: %s", roundID, err)
	}

	topology := r.GetTopology()
	nodeId := instance.GetID()

	// Pull the particular server host object from the commManager
	recipientID := topology.GetNextNode(nodeId)
	nextNodeIndex := topology.GetNodeLocation(recipientID)
	recipient := topology.GetHostAtIndex(nextNodeIndex)

	// Create the message structure to send the messages
	batch := &mixmessages.Batch{
		Round: &mixmessages.RoundInfo{
			ID: uint64(roundID),
		},
		FromPhase: int32(r.GetCurrentPhaseType()),
		Slots:     make([]*mixmessages.Slot, r.GetBatchSize()),
	}

	// For each message chunk (slot), fill the slots buffer
	// Note that this will panic if there are more slots than batchSize
	// (shouldn't be possible?)
	cnt := 0
	for chunk, finish := getChunk(); finish; chunk, finish = getChunk() {
		jww.ERROR.Printf("chunk end: %v", chunk.End())
		for i := chunk.Begin(); i < chunk.End(); i++ {
			msg := getMessage(i)
			batch.Slots[i] = msg
			cnt++
		}
	}

	jww.ERROR.Printf("went through %d times", cnt)
	localServer := instance.GetNetwork().String()
	port := strings.Split(localServer, ":")[1]
	addr := fmt.Sprintf("%s:%s", nodeId, port)
	name := services.NameStringer(addr, topology.GetNodeLocation(nodeId), topology.Len())

	measureFunc := r.GetCurrentPhase().Measure
	if measureFunc != nil {
		measureFunc(measure.TagTransmitLastSlot)
	}
	jww.INFO.Printf("[%s]: RID %d TransmitPhase FOR \"%s\" COMPLETE/SEND",
		name, roundID, r.GetCurrentPhaseType())

	jww.INFO.Printf("batch: %v", batch)
	// Make sure the comm doesn't return an Ack with an error message
	ack, err := instance.GetNetwork().SendPostPhase(recipient, batch)
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
