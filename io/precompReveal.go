////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package io

import (
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/comms/mixserver/message"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/cryptops/precomputation"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
)

// Blank struct for implementing services.BatchTransmission
type PrecompRevealHandler struct{}

// ReceptionHandler for PrecompRevealMessages
func (s ServerImpl) PrecompReveal(input *pb.PrecompRevealMessage) {
	jww.DEBUG.Printf("Received PrecompReveal Message %v from phase %s...",
		input.RoundID, globals.Phase(input.LastOp).String())
	// Get the input channel for the cryptop
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
}

// Convert the Reveal message to a Strip message and send to the last node
func precompRevealLastNode(roundId string, batchSize uint64,
	input *pb.PrecompRevealMessage) {
	jww.INFO.Println("Beginning PrecompStrip Phase...")
	// Create the SlotStripIn for sending into PrecompStrip
	stripChannel := globals.GlobalRoundMap.GetRound(roundId).GetChannel(
		globals.PRECOMP_STRIP)
	for i := uint64(0); i < batchSize; i++ {
		out := input.Slots[i]
		// Convert to SlotStripIn
		var slot services.Slot = &precomputation.PrecomputationSlot{
			Slot:              out.Slot,
			MessagePrecomputation:     cyclic.NewIntFromBytes(
				out.PartialMessageCypherText),
			RecipientIDPrecomputation: cyclic.NewIntFromBytes(
				out.PartialRecipientCypherText),
		}
		// Pass slot as input to Strip's channel
		stripChannel <- &slot
	}
}

// TransmissionHandler for PrecompRevealMessages
func (h PrecompRevealHandler) Handler(
	roundId string, batchSize uint64, slots []*services.Slot) {
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

	if IsLastNode {
		// Transition to PrecompStrip phase
		precompRevealLastNode(roundId, batchSize, msg)
	} else {
		// Send the completed PrecompRevealMessage
		jww.DEBUG.Printf("Sending PrecompReveal Message to %v...", NextServer)
		message.SendPrecompReveal(NextServer, msg)
	}
}
