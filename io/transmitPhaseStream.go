////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package io

// transmitPhaseStream.go contains the logic for streaming a phase comm

import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/primitives/id"
	"io"
	"strings"
	"time"
)

// StreamTransmitPhase streams slot messages to the provided Node.
func StreamTransmitPhase(roundID id.Round, serverInstance phase.GenericInstance, getChunk phase.GetChunk,
	getMessage phase.GetMessage) error {

	instance, ok := serverInstance.(*internal.Instance)
	if !ok {
		return errors.Errorf("Invalid server instance passed in")
	}
	//get the round so you can get its batch size
	r, err := instance.GetRoundManager().GetRound(roundID)
	if err != nil {
		return errors.Errorf("Could not retrieve round %d from"+
			" manager %s", roundID, err)
	}
	rType := r.GetCurrentPhaseType()
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

	// get the current phase
	// get this here to use down below to record the measurement to stop a race
	// conditions where other nodes finish their works and get this node to
	// iterate phase before the measure code runs
	currentPhase := r.GetCurrentPhase()

	// This gets the streaming client which used to send slots
	// using the recipient node id and the batch info header
	// It's context must be canceled after receiving an ack
	streamClient, cancel, err := instance.GetNetwork().GetPostPhaseStreamClient(
		recipient, header)
	if err != nil {
		return errors.Errorf("Error on comm, unable to get streaming "+
			"client: %+v", err)
	}
	defer cancel()

	//pull the first chunk reception out so that it can be timestamped
	chunk, finish := getChunk()
	var start time.Time
	numSlots := 0
	// For each message chunk (slot) stream it out
	for ; finish; chunk, finish = getChunk() {
		for i := chunk.Begin(); i < chunk.End(); i++ {
			numSlots++
			if numSlots == 1 {
				start = time.Now()
			}
			msg := getMessage(i)
			err = streamClient.Send(msg)
			if err != nil {
				eofAck, eofErr := streamClient.CloseAndRecv()
				if eofErr != nil {
					err = errors.Wrap(err, eofErr.Error())
				} else {
					err = errors.Wrap(err, eofAck.Error)
				}
				return errors.Errorf("Error on comm, not able to send "+
					"slot: %+v", err)
			}
		}
	}
	end := time.Now()
	measureFunc := currentPhase.Measure
	if measureFunc != nil {
		measureFunc(measure.TagTransmitLastSlot)
	}

	// Receive ack and cancel client streaming context
	ack, err := streamClient.CloseAndRecv()
	localServer := instance.GetNetwork().String()
	port := strings.Split(localServer, ":")[1]
	addr := fmt.Sprintf("%s:%s", nodeID, port)
	name := services.NameStringer(addr, topology.GetNodeLocation(nodeID),
		topology.Len())

	jww.INFO.Printf("[%s] RID %d StreamTransmitPhase FOR \"%s\""+
		" COMPLETE/SEND", name, roundID, rType)

	jww.INFO.Printf("\tbwLogging: Round %d, "+
		"transmitted phase: %s, "+
		"from: %s, to: %s, "+
		"started: %v, "+
		"ended: %v, "+
		"duration: %d,",
		roundID, currentPhase.GetType(),
		instance.GetID(), recipientID,
		start, end, end.Sub(start).Milliseconds())

	if err != nil {
		return errors.WithMessagef(err, "Failed to stream on round %d to %s",
			roundID, recipient.GetId())
	}

	// Make sure the comm doesn't return an Ack with an error message
	if ack != nil && ack.Error != "" {
		return errors.Errorf("Remote Server Error: %s", ack.Error)
	}

	return nil
}

// StreamPostPhase implements the server gRPC handler for receiving a
// phase from another node and sending the data into the Phase
func StreamPostPhase(p phase.Phase, batchSize uint32,
	stream mixmessages.Node_StreamPostPhaseServer) (*streamInfo, error) {
	// Send a chunk for each slot received along with
	// its index until all slots or an error is received
	slot, err := stream.Recv()
	var start, end time.Time
	slotsReceived := uint32(0)
	for ; err == nil; slot, err = stream.Recv() {
		slotsReceived++

		if slotsReceived == 1 {
			start = time.Now()
		}
		index := slot.Index

		// Input the slot into the current Phase
		phaseErr := p.Input(index, slot)
		if phaseErr != nil {
			err = errors.Errorf("Failed on phase input %v for slot %v: %+v",
				index, slot, phaseErr)
			return &streamInfo{Start: start, End: end}, phaseErr
		}

		// Build a Chunk corresponding to the slot and send it to the Phase
		chunk := services.NewChunk(index, index+1)
		p.Send(chunk)

		if slotsReceived == batchSize {
			end = time.Now()
		}
	}

	// Set error in ack message if we didn't receive all slots
	ack := messages.Ack{
		Error: "",
	}
	if err != io.EOF {
		ack.Error = fmt.Sprintf("errors occurred, %v/%v slots "+
			"recived: %s", slotsReceived, batchSize, err.Error())
	} else if slotsReceived != batchSize {
		ack.Error = fmt.Sprintf("Mismatch between batch size %v"+
			"and received num slots %v, no error", slotsReceived, batchSize)
	}

	// Close the stream by sending ack
	// and returning whether it succeeded
	errClose := stream.SendAndClose(&ack)

	si := &streamInfo{Start: start, End: end}

	if errClose != nil && ack.Error != "" {
		return si, errors.WithMessage(errClose, ack.Error)
	} else if errClose == nil && ack.Error != "" {
		return si, errors.New(ack.Error)
	} else {
		return si, errClose
	}
}
