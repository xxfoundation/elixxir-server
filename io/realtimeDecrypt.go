////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/id"
	"gitlab.com/elixxir/server/cryptops/realtime"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/services"
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

	// MARK To test the rounds timing out, switch which line is commented
	timeoutRealtime(input.RoundID, 10*time.Minute)
	//timeoutRealtime(input.RoundID, 20*time.Millisecond)

	// Get the input channel for the cryptop
	chIn := s.GetChannel(input.RoundID, globals.REAL_DECRYPT)

	// Store when the operation started
	globals.GlobalRoundMap.GetRound(input.RoundID).CryptopStartTimes[globals.
		REAL_DECRYPT] = startTime

	// Iterate through the Slots in the RealtimeDecryptMessage
	for i := 0; i < len(input.Slots); i++ {
		// Convert input message to equivalent SlotDecrypt
		in := input.Slots[i]
		userId := new(id.UserID).SetBytes(in.SenderID)
		var slot services.Slot = &realtime.Slot{
			Slot:               uint64(i),
			CurrentID:          userId,
			Message:            cyclic.NewIntFromBytes(in.MessagePayload),
			EncryptedRecipient: cyclic.NewIntFromBytes(in.RecipientID),
			CurrentKey:         cyclic.NewMaxInt(),
			Salt:               in.Salt,
			// TODO: How will we pass and verify the kmac?
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

	// Store when the operation started
	globals.GlobalRoundMap.GetRound(input.RoundID).CryptopStartTimes[globals.
		REAL_DECRYPT] = startTime

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
			Slot:                 uint64(i),
			EncryptedMessage:     out.MessagePayload,
			EncryptedRecipientID: out.RecipientID,
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
	roundID string, batchSize uint64, slots []*services.Slot) {
	startTime := time.Now()
	jww.INFO.Printf("Starting RealtimeDecrypt.Handler(RoundId: %s) at %s",
		roundID, startTime.Format(time.RFC3339))

	elapsed := startTime.Sub(globals.GlobalRoundMap.GetRound(roundID).
		CryptopStartTimes[globals.REAL_DECRYPT])

	jww.DEBUG.Printf("RealtimeDecrypt Crypto took %v ms for "+
		"RoundId %s", elapsed, roundID)

	// Create the RealtimeDecryptMessage
	msg := &pb.RealtimeDecryptMessage{
		RoundID: roundID,
		LastOp:  int32(globals.REAL_DECRYPT),
		Slots:   make([]*pb.CmixMessage, batchSize),
	}

	// Iterate over the output channel
	for i := uint64(0); i < batchSize; i++ {
		// Type assert Slot to SlotDecrypt
		out := (*slots[i]).(*realtime.Slot)
		// Convert to CmixMessage
		msgSlot := &pb.CmixMessage{
			SenderID:       out.CurrentID[:],
			MessagePayload: out.Message.Bytes(),
			RecipientID:    out.EncryptedRecipient.Bytes(),
			Salt:           out.Salt,
		}

		// Append the CmixMessage to the RealtimeDecryptMessage
		msg.Slots[out.Slot] = msgSlot
	}

	// Advance internal state to the next phase
	globals.GlobalRoundMap.SetPhase(roundID, globals.REAL_PERMUTE)

	sendTime := time.Now()
	if globals.IsLastNode {
		// Transition to RealtimePermute phase
		jww.INFO.Printf("Starting RealtimePermute  Phase to %v at %s",
			NextServer, sendTime.Format(time.RFC3339))
		realtimeDecryptLastNode(roundID, batchSize, msg)
	} else {
		// Send the completed RealtimeDecryptMessage
		jww.INFO.Printf("Sending RealtimeDecrypt Message to %v at %s",
			NextServer, sendTime.Format(time.RFC3339))
		node.SendRealtimeDecrypt(NextServer, msg)
	}

	endTime := time.Now()
	jww.INFO.Printf("Finished RealtimeDecrypt.Handler(RoundId: %s) in %d ms",
		roundID, (endTime.Sub(startTime))/time.Millisecond)
}

// Kickoff for RealtimeDecryptMessages
// TODO Remove this duplication
func KickoffDecryptHandler(roundID string, batchSize uint64,
	slots []*realtime.Slot) {
	startTime := time.Now()
	jww.INFO.Printf("[Last Node] Starting KickoffDecryptHandler(RoundId: %s)"+
		" at %s",
		roundID, startTime.Format(time.RFC3339))

	// Create the RealtimeDecryptMessage
	msg := &pb.RealtimeDecryptMessage{
		RoundID: roundID,
		LastOp:  int32(globals.PRECOMP_COMPLETE),
		Slots:   make([]*pb.CmixMessage, batchSize),
	}

	// Iterate over the output channel
	for i := uint64(0); i < batchSize; i++ {
		out := slots[i]
		msgSlot := &pb.CmixMessage{
			SenderID:       out.CurrentID[:],
			MessagePayload: out.Message.Bytes(),
			RecipientID:    out.EncryptedRecipient.Bytes(),
			Salt:           out.Salt,
		}

		// Append the CmixMessage to the RealtimeDecryptMessage
		msg.Slots[out.Slot] = msgSlot
	}

	// Advance internal state to the next phase
	globals.GlobalRoundMap.SetPhase(roundID, globals.REAL_DECRYPT)

	// Send the completed RealtimeDecryptMessage
	sendTime := time.Now()
	jww.INFO.Printf("Sending RealtimeDecrypt Message to %v at %s",
		NextServer, sendTime.Format(time.RFC3339))
	node.SendRealtimeDecrypt(NextServer, msg)

	endTime := time.Now()
	jww.INFO.Printf("[Last Node] Finished KickoffDecryptHandler(RoundId: %s)"+
		" in %d ms",
		roundID, (endTime.Sub(startTime))/time.Millisecond)
}
