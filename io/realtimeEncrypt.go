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
	"gitlab.com/privategrity/crypto/id"
)

// Blank struct for implementing services.BatchTransmission
type RealtimeEncryptHandler struct{}

// ReceptionHandler for RealtimeEncryptMessages
func (s ServerImpl) RealtimeEncrypt(input *pb.RealtimeEncryptMessage) {
	startTime := time.Now()
	jww.INFO.Printf("Starting RealtimeEncrypt(RoundId: %s, Phase: %s) at %s",
		input.RoundID, globals.Phase(input.LastOp).String(),
		startTime.Format(time.RFC3339))
	// Get the input channel for the cryptop
	chIn := s.GetChannel(input.RoundID, globals.REAL_ENCRYPT)

	// Store when the operation started
	globals.GlobalRoundMap.GetRound(input.RoundID).CryptopStartTimes[globals.
		REAL_ENCRYPT] = startTime

	// Iterate through the Slots in the RealtimeEncryptMessage
	for i := 0; i < len(input.Slots); i++ {
		// Convert input message to equivalent SlotEncrypt
		in := input.Slots[i]
		userId, err := new(id.UserID).SetBytes(in.RecipientID)
		if err != nil {
			jww.ERROR.Printf("RealtimeEncrypt: Couldn't populate user ID from" +
				" bytes: %v", err.Error())
		}
		var slot services.Slot = &realtime.Slot{
			Slot:       uint64(i),
			CurrentID:  userId,
			Message:    cyclic.NewIntFromBytes(in.MessagePayload),
			CurrentKey: cyclic.NewMaxInt(),
			Salt:       in.Salt,
		}
		// Pass slot as input to Encrypt's channel
		chIn <- &slot
	}

	close(chIn)

	endTime := time.Now()
	jww.INFO.Printf("Finished RealtimeEncrypt(RoundId: %s, Phase: %s) in %d ms",
		input.RoundID, globals.Phase(input.LastOp).String(),
		(endTime.Sub(startTime))/time.Millisecond)
}

// Transition to RealtimePeel phase on the last node
func realtimeEncryptLastNode(roundID string, batchSize uint64,
	input *pb.RealtimeEncryptMessage) {

	startTime := time.Now()
	jww.INFO.Printf("[Last Node] Initializing RealtimePeel(RoundId: %s, "+
		"Phase: %s) at %s",
		input.RoundID, globals.Phase(input.LastOp).String(),
		startTime.Format(time.RFC3339))

	// Store when the operation started
	globals.GlobalRoundMap.GetRound(input.RoundID).CryptopStartTimes[globals.
		REAL_ENCRYPT] = startTime

	// TODO: record the start precomp permute time for this round here,
	//       and print the time it took for the Decrypt phase to complete.

	// Get round and channel
	round := globals.GlobalRoundMap.GetRound(roundID)
	if round == nil {
		jww.INFO.Printf("skipping round %s, because it's dead", roundID)
		return
	}

	peelChannel := round.GetChannel(globals.REAL_PEEL)
	// Create the Slot for sending into RealtimePeel

	// Store when the operation started
	globals.GlobalRoundMap.GetRound(input.RoundID).CryptopStartTimes[globals.
		REAL_PEEL] = time.Now()

	for i := uint64(0); i < batchSize; i++ {
		out := input.Slots[i]
		// Convert to Slot
		userId, err := new(id.UserID).SetBytes(out.RecipientID)
		if err != nil {
			jww.ERROR.Printf("RealtimeEncryptLastNode: Couldn't create user" +
				" ID from bytes: %v", err.Error())
		}
		var slot services.Slot = &realtime.Slot{
			Slot:       i,
			CurrentID:  userId,
			Message:    cyclic.NewIntFromBytes(out.MessagePayload),
			CurrentKey: cyclic.NewMaxInt(),
			Salt:       out.Salt,
		}
		// Pass slot as input to Peel's channel
		peelChannel <- &slot
	}

	endTime := time.Now()
	jww.INFO.Printf("[Last Node] Finished Initializing "+
		"RealtimePeel(RoundId: %s, Phase: %s) in %d ms",
		input.RoundID, globals.Phase(input.LastOp).String(),
		(endTime.Sub(startTime))/time.Millisecond)
}

// TransmissionHandler for RealtimeEncryptMessages
func (h RealtimeEncryptHandler) Handler(
	roundID string, batchSize uint64, slots []*services.Slot) {
	startTime := time.Now()
	jww.INFO.Printf("Starting RealtimeEncrypt.Handler(RoundId: %s) at %s",
		roundID, startTime.Format(time.RFC3339))

	elapsed := startTime.Sub(globals.GlobalRoundMap.GetRound(roundID).
		CryptopStartTimes[globals.REAL_ENCRYPT])

	jww.DEBUG.Printf("RealtimeEncrypt Crypto took %v ms for "+
		"RoundId %s", elapsed, roundID)

	// Create the RealtimeEncryptMessage
	msg := &pb.RealtimeEncryptMessage{
		RoundID: roundID,
		LastOp:  int32(globals.REAL_ENCRYPT),
		Slots:   make([]*pb.CmixMessage, batchSize),
	}

	// Iterate over the output channel
	for i := uint64(0); i < batchSize; i++ {
		// Type assert Slot to SlotEncrypt
		out := (*slots[i]).(*realtime.Slot)
		// Convert to CmixMessage
		msgSlot := &pb.CmixMessage{
			SenderID:       id.ZeroID[:],
			RecipientID:    out.CurrentID[:],
			MessagePayload: out.Message.Bytes(),
			Salt:           out.Salt,
		}

		// Append the CmixMessage to the RealtimeEncryptMessage
		msg.Slots[out.Slot] = msgSlot
	}

	sendTime := time.Now()
	if globals.IsLastNode {
		// Transition to RealtimePeel phase
		jww.INFO.Printf("Starting RealtimePeel Phase to %v at %s",
			NextServer, sendTime.Format(time.RFC3339))
		// Advance internal state to the next phase
		globals.GlobalRoundMap.SetPhase(roundID, globals.REAL_PEEL)
		realtimeEncryptLastNode(roundID, batchSize, msg)
	} else {
		// Send the completed RealtimeEncryptMessage
		jww.INFO.Printf("Sending RealtimeEncrypt Message to %v at %s",
			NextServer, sendTime.Format(time.RFC3339))
		// Advance internal state to the next phase
		globals.GlobalRoundMap.SetPhase(roundID, globals.REAL_COMPLETE)
		node.SendRealtimeEncrypt(NextServer, msg)
		globals.GlobalRoundMap.DeleteRound(roundID)
	}

	endTime := time.Now()
	jww.INFO.Printf("Finished RealtimeEncrypt.Handler(RoundId: %s) in %d ms",
		roundID, (endTime.Sub(startTime))/time.Millisecond)
}
