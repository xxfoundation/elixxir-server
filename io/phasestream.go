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
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/services"
	"io"
	"strings"
)

// StreamTransmitPhase streams slot messages to the provided Node.
func StreamTransmitPhase(roundID id.Round, serverInstance phase.GenericInstance, getChunk phase.GetChunk,
	getMessage phase.GetMessage) error {

	instance, ok := serverInstance.(*server.Instance)
	if !ok {
		return errors.Errorf("Invalid server instance passed in")
	}
	//get the round so you can get its batch size
	r, err := instance.GetRoundManager().GetRound(roundID)
	if err != nil {
		return errors.Errorf("Could not retrieve round %d from manager  %s", roundID, err)
	}

	topology := r.GetTopology()
	nodeID := instance.GetID()

	// Pull the particular server host object from the commManager
	recipientID := topology.GetNextNode(nodeID)
	recipientIndex := topology.GetNodeLocation(recipientID)
	recipient := topology.GetHostAtIndex(recipientIndex)

	header := mixmessages.BatchInfo{
		Round: &mixmessages.RoundInfo{
			ID: uint64(roundID),
		},
		FromPhase: int32(r.GetCurrentPhaseType()),
		BatchSize: r.GetBatchSize(),
	}

	// This gets the streaming client which used to send slots
	// using the recipient node id and the batch info header
	// It's context must be canceled after receiving an ack
	streamClient, cancel, err := instance.GetNetwork().GetPostPhaseStreamClient(
		recipient, header)
	if err != nil {
		return errors.Errorf("Error on comm, unable to get streaming client: %+v",
			err)
	}

	// For each message chunk (slot) stream it out
	for chunk, finish := getChunk(); finish; chunk, finish = getChunk() {
		for i := chunk.Begin(); i < chunk.End(); i++ {
			msg := getMessage(i)
			err := streamClient.Send(msg)
			if err != nil {
				return errors.Errorf("Error on comm, not able to send slot: %+v",
					err)
			}
		}
	}

	measureFunc := r.GetCurrentPhase().Measure
	if measureFunc != nil {
		measureFunc(measure.TagTransmitLastSlot)
	}

	// Receive ack and cancel client streaming context
	ack, err := streamClient.CloseAndRecv()
	localServer := instance.GetNetwork().String()
	port := strings.Split(localServer, ":")[1]
	addr := fmt.Sprintf("%s:%s", nodeID, port)
	name := services.NameStringer(addr, topology.GetNodeLocation(nodeID), topology.Len())

	jww.INFO.Printf("[%s] RID %d StreamTransmitPhase FOR \"%s\" COMPLETE/SEND",
		name, roundID, r.GetCurrentPhaseType())

	cancel()

	if err != nil {
		return err
	}

	// Make sure the comm doesn't return an Ack with an error message
	if ack != nil && ack.Error != "" {
		return errors.Errorf("Remote Server Error: %s", ack.Error)
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
			err = errors.Errorf("Failed on phase input %v for slot %v: %+v",
				index, slot, phaseErr)
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
		err = errors.Errorf("Mismatch between batch size %v"+
			"and received num slots %v", batchSize, slotsReceived)
		return err
	}

	// Close the stream by sending ack
	// and returning whether it succeeded
	return stream.SendAndClose(&ack)
}
