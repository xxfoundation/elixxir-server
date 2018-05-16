////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/comms/node"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/cryptops/realtime"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
	"time"
)

// Blank struct for implementing services.BatchTransmission
type RealtimeDecryptHandler struct{}

// ReceptionHandler for RealtimeDecryptMessages
func (s ServerImpl) RealtimeDecrypt(input *pb.RealtimeDecryptMessage) {
	startTime := time.Now()
	jww.INFO.Printf("Starting RealtimeDecrypt(RoundId: %s, Phase: %s) at %s",
		input.RoundID, globals.Phase(input.LastOp).String(),
		startTime.Format(time.RFC3339))

	timeoutRealtime(input.RoundID, 10*time.Minute)

	// Get the input channel for the cryptop
	chIn := s.GetChannel(input.RoundID, globals.REAL_DECRYPT)
	// Iterate through the Slots in the RealtimeDecryptMessage
	for i := 0; i < len(input.Slots); i++ {
		// Convert input message to equivalent SlotDecrypt
		in := input.Slots[i]
		var slot services.Slot = &realtime.RealtimeSlot{
			Slot:               in.Slot,
			CurrentID:          in.SenderID,
			Message:            cyclic.NewIntFromBytes(in.EncryptedMessage),
			EncryptedRecipient: cyclic.NewIntFromBytes(in.EncryptedRecipientID),
			CurrentKey:         cyclic.NewInt(1),
		}
		// Pass slot as input to Decrypt's channel
		chIn <- &slot
	}

	close(chIn)

	endTime := time.Now()
	jww.INFO.Printf("Finished RealtimeDecrypt(RoundId: %s, Phase: %s) in %d ms",
		input.RoundID, globals.Phase(input.LastOp).String(),
		(endTime.Sub(startTime))/time.Millisecond)
}

// Transition to RealtimePermute phase on the last node
func realtimeDecryptLastNode(roundId string, batchSize uint64,
	input *pb.RealtimeDecryptMessage) {

	startTime := time.Now()
	jww.INFO.Printf("[Last Node] Initializing RealtimePermute(RoundId: %s, "+
		"Phase: %s) at %s",
		input.RoundID, globals.Phase(input.LastOp).String(),
		startTime.Format(time.RFC3339))

	// TODO: record the start precomp permute time for this round here,
	//       and print the time it took for the Decrypt phase to complete.

	// Create the RealtimePermuteMessage
	msg := &pb.RealtimePermuteMessage{
		RoundID: roundId,
		LastOp:  int32(globals.REAL_DECRYPT),
		Slots:   make([]*pb.RealtimePermuteSlot, batchSize),
	}

	// Iterate over the input slots
	for i := range input.Slots {
		out := input.Slots[i]
		// Convert to RealtimePermuteSlot
		msgSlot := &pb.RealtimePermuteSlot{
			Slot:                 out.Slot,
			EncryptedMessage:     out.EncryptedMessage,
			EncryptedRecipientID: out.EncryptedRecipientID,
		}

		// Append the RealtimePermuteSlot to the RealtimePermuteMessage
		msg.Slots[i] = msgSlot
	}

	// Send the first RealtimePermute Message
	sendTime := time.Now()
	jww.INFO.Printf("[Last Node] Sending RealtimePermute Messages to %v at %s",
		NextServer, sendTime.Format(time.RFC3339))
	node.SendRealtimePermute(NextServer, msg)

	endTime := time.Now()
	jww.INFO.Printf("[Last Node] Finished Initializing "+
		"RealtimePermute(RoundId: %s, Phase: %s) in %d ms",
		input.RoundID, globals.Phase(input.LastOp).String(),
		(endTime.Sub(startTime))/time.Millisecond)
}

// TransmissionHandler for RealtimeDecryptMessages
func (h RealtimeDecryptHandler) Handler(
	roundId string, batchSize uint64, slots []*services.Slot) {
	startTime := time.Now()
	jww.INFO.Printf("Starting RealtimeDecrypt.Handler(RoundId: %s) at %s",
		roundId, startTime.Format(time.RFC3339))

	// Create the RealtimeDecryptMessage
	msg := &pb.RealtimeDecryptMessage{
		RoundID: roundId,
		LastOp:  int32(globals.REAL_DECRYPT),
		Slots:   make([]*pb.RealtimeDecryptSlot, batchSize),
	}

	// Iterate over the output channel
	for i := uint64(0); i < batchSize; i++ {
		// Type assert Slot to SlotDecrypt
		out := (*slots[i]).(*realtime.RealtimeSlot)
		// Convert to RealtimeDecryptSlot
		msgSlot := &pb.RealtimeDecryptSlot{
			Slot:                 out.Slot,
			SenderID:             out.CurrentID,
			EncryptedMessage:     out.Message.Bytes(),
			EncryptedRecipientID: out.EncryptedRecipient.Bytes(),
		}

		// Append the RealtimeDecryptSlot to the RealtimeDecryptMessage
		msg.Slots[i] = msgSlot
	}

	// Advance internal state to the next phase
	globals.GlobalRoundMap.GetRound(roundId).SetPhase(globals.REAL_PERMUTE)

	sendTime := time.Now()
	if globals.IsLastNode {
		// Transition to RealtimePermute phase
		jww.INFO.Printf("Starting RealtimePermute  Phase to %v at %s",
			NextServer, sendTime.Format(time.RFC3339))
		realtimeDecryptLastNode(roundId, batchSize, msg)
	} else {
		// Send the completed RealtimeDecryptMessage
		jww.INFO.Printf("Sending RealtimeDecrypt Message to %v at %s",
			NextServer, sendTime.Format(time.RFC3339))
		node.SendRealtimeDecrypt(NextServer, msg)
	}

	endTime := time.Now()
	jww.INFO.Printf("Finished RealtimeDecrypt.Handler(RoundId: %s) in %d ms",
		roundId, (endTime.Sub(startTime))/time.Millisecond)
}

// Kickoff for RealtimeDecryptMessages
// TODO Remove this duplication
func KickoffDecryptHandler(roundId string, batchSize uint64,
	slots []*realtime.RealtimeSlot) {
	startTime := time.Now()
	jww.INFO.Printf("[Last Node] Starting KickoffDecryptHandler(RoundId: %s)"+
		" at %s",
		roundId, startTime.Format(time.RFC3339))

	// Create the RealtimeDecryptMessage
	msg := &pb.RealtimeDecryptMessage{
		RoundID: roundId,
		LastOp:  int32(globals.PRECOMP_COMPLETE),
		Slots:   make([]*pb.RealtimeDecryptSlot, batchSize),
	}

	// Iterate over the output channel
	for i := uint64(0); i < batchSize; i++ {
		out := slots[i]
		msgSlot := &pb.RealtimeDecryptSlot{
			Slot:                 out.Slot,
			SenderID:             out.CurrentID,
			EncryptedMessage:     out.Message.Bytes(),
			EncryptedRecipientID: out.EncryptedRecipient.Bytes(),
		}

		// Append the RealtimeDecryptSlot to the RealtimeDecryptMessage
		msg.Slots[i] = msgSlot
	}

	// Advance internal state to the next phase
	globals.GlobalRoundMap.GetRound(roundId).SetPhase(globals.REAL_DECRYPT)

	// Send the completed RealtimeDecryptMessage
	sendTime := time.Now()
	jww.INFO.Printf("Sending RealtimeDecrypt Message to %v at %s",
		NextServer, sendTime.Format(time.RFC3339))
	node.SendRealtimeDecrypt(NextServer, msg)

	endTime := time.Now()
	jww.INFO.Printf("[Last Node] Finished KickoffDecryptHandler(RoundId: %s)"+
		" in %d ms",
		roundId, (endTime.Sub(startTime))/time.Millisecond)
}
