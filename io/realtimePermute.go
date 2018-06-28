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
type RealtimePermuteHandler struct{}

// ReceptionHandler for RealtimePermuteMessages
func (s ServerImpl) RealtimePermute(input *pb.RealtimePermuteMessage) {
	startTime := time.Now()
	jww.INFO.Printf("Starting RealtimePermute(RoundId: %s, Phase: %s) at %s",
		input.RoundID, globals.Phase(input.LastOp).String(),
		startTime.Format(time.RFC3339))

	// Store when the operation started
	globals.GlobalRoundMap.GetRound(input.RoundID).CryptopStartTimes[globals.
		REAL_PERMUTE] = startTime

	// Get the input channel for the cryptop
	chIn := s.GetChannel(input.RoundID, globals.REAL_PERMUTE)
	// Iterate through the Slots in the RealtimePermuteMessage
	for i := 0; i < len(input.Slots); i++ {
		// Convert input message to equivalent Slot
		in := input.Slots[i]
		var slot services.Slot = &realtime.Slot{
			Slot: in.Slot,
			Message: cyclic.NewIntFromBytes(
				in.EncryptedMessage),
			EncryptedRecipient: cyclic.NewIntFromBytes(
				in.EncryptedRecipientID),
		}
		// Pass slot as input to Permute's channel
		chIn <- &slot
	}

	close(chIn)

	endTime := time.Now()
	jww.INFO.Printf("Finished RealtimePermute(RoundId: %s, Phase: %s) in %d ms",
		input.RoundID, globals.Phase(input.LastOp).String(),
		(endTime.Sub(startTime))/time.Millisecond)
}

// Transition to RealtimeIdentify phase on the last node
func realtimePermuteLastNode(roundId string, batchSize uint64,
	input *pb.RealtimePermuteMessage) {

	startTime := time.Now()
	jww.INFO.Printf("[Last Node] Initializing RealtimeIdentify(RoundId: %s, "+
		"Phase: %s) at %s",
		input.RoundID, globals.Phase(input.LastOp).String(),
		startTime.Format(time.RFC3339))

	// Store when the operation started
	globals.GlobalRoundMap.GetRound(input.RoundID).CryptopStartTimes[globals.
		REAL_PERMUTE] = startTime

	// TODO: record the start time for this round here,
	//       and print the time it took for the last phase to complete.

	// Get round and channel
	round := globals.GlobalRoundMap.GetRound(roundId)
	if round == nil {
		jww.INFO.Printf("skipping round %s, because it's dead", roundId)
		return
	}

	identifyChannel := round.GetChannel(globals.REAL_IDENTIFY)

	// Store when the operation started
	globals.GlobalRoundMap.GetRound(input.RoundID).CryptopStartTimes[globals.
		REAL_IDENTIFY] = time.Now()

	// Create the Slot for sending into RealtimeIdentify
	for i := uint64(0); i < batchSize; i++ {
		out := input.Slots[i]
		// Convert to Slot
		var slot services.Slot = &realtime.Slot{
			Slot:               out.Slot,
			EncryptedRecipient: cyclic.NewIntFromBytes(out.EncryptedRecipientID),
		}
		// Save EncryptedMessages for the Identify->Encrypt transition
		round.LastNode.EncryptedMessage[i] = cyclic.NewIntFromBytes(out.EncryptedMessage)
		// Pass slot as input to Identify's channel
		identifyChannel <- &slot
	}

	endTime := time.Now()
	jww.INFO.Printf("[Last Node] Finished Initializing "+
		"RealtimeIdentify(RoundId: %s, Phase: %s) in %d ms",
		input.RoundID, globals.Phase(input.LastOp).String(),
		(endTime.Sub(startTime))/time.Millisecond)
}

// TransmissionHandler for RealtimePermuteMessages
func (h RealtimePermuteHandler) Handler(
	roundId string, batchSize uint64, slots []*services.Slot) {
	startTime := time.Now()
	jww.INFO.Printf("Starting RealtimePermute.Handler(RoundId: %s) at %s",
		roundId, startTime.Format(time.RFC3339))

	elapsed := startTime.Sub(globals.GlobalRoundMap.GetRound(roundId).
		CryptopStartTimes[globals.REAL_PERMUTE])

	jww.DEBUG.Printf("RealtimePermute Crypto took %v ms for "+
		"RoundId %s", elapsed, roundId)

	// Create the RealtimePermuteMessage for sending
	msg := &pb.RealtimePermuteMessage{
		RoundID: roundId,
		LastOp:  int32(globals.REAL_PERMUTE),
		Slots:   make([]*pb.RealtimePermuteSlot, batchSize),
	}

	// Iterate over the output channel
	for i := uint64(0); i < batchSize; i++ {
		// Type assert Slot to Slot
		out := (*slots[i]).(*realtime.Slot)
		// Convert to RealtimePermuteSlot
		msgSlot := &pb.RealtimePermuteSlot{
			Slot:                 out.Slot,
			EncryptedMessage:     out.Message.Bytes(),
			EncryptedRecipientID: out.EncryptedRecipient.Bytes(),
		}

		// Append the RealtimePermuteSlot to the RealtimePermuteMessage
		msg.Slots[i] = msgSlot
	}

	sendTime := time.Now()
	if globals.IsLastNode {
		// Transition to RealtimeIdentify phase
		// Advance internal state to the next phase
		globals.GlobalRoundMap.SetPhase(roundId, globals.REAL_IDENTIFY)
		jww.INFO.Printf("Starting RealtimeIdentify Phase to %v at %s",
			NextServer, sendTime.Format(time.RFC3339))
		realtimePermuteLastNode(roundId, batchSize, msg)
	} else {
		// Send the completed RealtimePermuteMessage
		// Advance internal state to the next phase
		globals.GlobalRoundMap.SetPhase(roundId, globals.REAL_ENCRYPT)
		jww.INFO.Printf("Sending RealtimePermute Message to %v at %s",
			NextServer, sendTime.Format(time.RFC3339))
		node.SendRealtimePermute(NextServer, msg)
	}

	endTime := time.Now()
	jww.INFO.Printf("Finished RealtimePermute.Handler(RoundId: %s) in %d ms",
		roundId, (endTime.Sub(startTime))/time.Millisecond)
}
