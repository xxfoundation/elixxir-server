////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package io

import (
	"gitlab.com/privategrity/comms/node"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/cryptops/precomputation"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"

	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"time"
)

// Blank struct for implementing services.BatchTransmission
type PrecompRevealHandler struct{}

// ReceptionHandler for PrecompRevealMessages
func (s ServerImpl) PrecompReveal(input *pb.PrecompRevealMessage) {
	startTime := time.Now()
	jww.INFO.Printf("Starting PrecompReveal(RoundId: %s, Phase: %s) at %s",
		input.RoundID, globals.Phase(input.LastOp).String(),
		startTime.Format(time.RFC3339))

	// Get the input channel for the cryptop
	defer recoverSetPhasePanic(input.RoundID)
	chIn := s.GetChannel(input.RoundID, globals.PRECOMP_REVEAL)
	// Iterate through the Slots in the PrecompRevealMessage
	for i := 0; i < len(input.Slots); i++ {
		// Convert input message to equivalent SlotReveal
		in := input.Slots[i]
		var slot services.Slot = &precomputation.PrecomputationSlot{
			Slot: in.Slot,
			MessagePrecomputation: cyclic.NewIntFromBytes(
				in.PartialMessageCypherText),
			RecipientIDPrecomputation: cyclic.NewIntFromBytes(
				in.PartialRecipientCypherText),
		}
		// Pass slot as input to Reveal's channel
		chIn <- &slot
	}

	close(chIn)

	endTime := time.Now()
	jww.INFO.Printf("Finished PrecompReveal(RoundId: %s, Phase: %s) in %d ms",
		input.RoundID, globals.Phase(input.LastOp).String(),
		(endTime.Sub(startTime))/time.Millisecond)
}

// Convert the Reveal message to a Strip message and send to the last node
func precompRevealLastNode(roundId string, batchSize uint64,
	input *pb.PrecompRevealMessage) {

	startTime := time.Now()
	jww.INFO.Printf("[Last Node] Initializing PrecompStrip(RoundId: %s, "+
		"Phase: %s) at %s",
		input.RoundID, globals.Phase(input.LastOp).String(),
		startTime.Format(time.RFC3339))

	// TODO: record the start precomp permute time for this round here,
	//       and print the time it took for the Decrypt phase to complete.

	// Create the SlotStripIn for sending into PrecompStrip
	stripChannel := globals.GlobalRoundMap.GetRound(roundId).GetChannel(
		globals.PRECOMP_STRIP)
	for i := uint64(0); i < batchSize; i++ {
		out := input.Slots[i]
		// Convert to SlotStripIn
		var slot services.Slot = &precomputation.PrecomputationSlot{
			Slot: out.Slot,
			MessagePrecomputation: cyclic.NewIntFromBytes(
				out.PartialMessageCypherText),
			RecipientIDPrecomputation: cyclic.NewIntFromBytes(
				out.PartialRecipientCypherText),
		}
		// Pass slot as input to Strip's channel
		stripChannel <- &slot
	}

	endTime := time.Now()
	jww.INFO.Printf("[Last Node] Finished Initializing "+
		"PrecompStrip(RoundId: %s, Phase: %s) in %d ms",
		input.RoundID, globals.Phase(input.LastOp).String(),
		(endTime.Sub(startTime))/time.Millisecond)
}

// TransmissionHandler for PrecompRevealMessages
func (h PrecompRevealHandler) Handler(
	roundId string, batchSize uint64, slots []*services.Slot) {
	startTime := time.Now()
	jww.INFO.Printf("Starting PrecompReveal.Handler(RoundId: %s) at %s",
		roundId, startTime.Format(time.RFC3339))

	// Create the PrecompRevealMessage
	msg := &pb.PrecompRevealMessage{
		RoundID: roundId,
		LastOp:  int32(globals.PRECOMP_REVEAL),
		Slots:   make([]*pb.PrecompRevealSlot, batchSize),
	}

	// Iterate over the output channel
	for i := uint64(0); i < batchSize; i++ {
		// Type assert Slot to SlotReveal
		out := (*slots[i]).(*precomputation.PrecomputationSlot)
		// Convert to PrecompRevealSlot
		msgSlot := &pb.PrecompRevealSlot{
			Slot: out.Slot,
			PartialMessageCypherText:   out.MessagePrecomputation.Bytes(),
			PartialRecipientCypherText: out.RecipientIDPrecomputation.Bytes(),
		}

		// Put it into the slice
		msg.Slots[i] = msgSlot
	}

	sendTime := time.Now()
	if globals.IsLastNode {
		// Transition to PrecompStrip phase
		// Advance internal state to the next phase
		defer recoverSetPhasePanic(roundId)
		globals.GlobalRoundMap.GetRound(roundId).SetPhase(globals.PRECOMP_STRIP)
		jww.INFO.Printf("Starting PrecompStrip Phase to %v at %s",
			NextServer, sendTime.Format(time.RFC3339))
		precompRevealLastNode(roundId, batchSize, msg)
	} else {
		// Send the completed PrecompRevealMessage
		jww.INFO.Printf("Sending PrecompReveal Message to %v at %s",
			NextServer, sendTime.Format(time.RFC3339))
		// Advance internal state to the next phase
		defer recoverSetPhasePanic(roundId)
		globals.GlobalRoundMap.GetRound(roundId).SetPhase(globals.REAL_DECRYPT)
		node.SendPrecompReveal(NextServer, msg)
	}

	endTime := time.Now()
	jww.INFO.Printf("Finished PrecompReveal.Handler(RoundId: %s) in %d ms",
		roundId, (endTime.Sub(startTime))/time.Millisecond)
}
