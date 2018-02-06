package io

import (
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
	outputChannel := s.GetChannel(input.RoundID, globals.PRECOMP_REVEAL)
	// Iterate through the Slots in the PrecompRevealMessage
	for i := 0; i < len(input.Slots); i++ {
		// Convert input message to equivalent SlotReveal
		in := input.Slots[i]
		var slot services.Slot = &precomputation.SlotReveal{
			Slot: in.Slot,
			PartialMessageCypherText: cyclic.NewIntFromBytes(
				in.PartialMessageCypherText),
		}
		// Pass slot as input to Reveal's channel
		outputChannel <- &slot
	}
}

// TransmissionHandler for PrecompRevealMessages
func (h PrecompRevealHandler) Handler(
	roundId string, batchSize uint64, slots []*services.Slot) {
	// Create the PrecompRevealMessage
	msg := &pb.PrecompRevealMessage{
		RoundID: roundId,
		Slots:   make([]*pb.PrecompRevealSlot, batchSize),
	}

	// Iterate over the output channel
	for i := uint64(0); i < batchSize; i++ {
		// Type assert Slot to SlotReveal
		out := (*slots[i]).(*precomputation.SlotReveal)
		// Convert to PrecompRevealSlot
		msgSlot := &pb.PrecompRevealSlot{
			Slot: out.Slot,
			PartialMessageCypherText: out.PartialMessageCypherText.Bytes(),
		}

		// Put it into the slice
		msg.Slots[i] = msgSlot
	}
	// Send the completed PrecompRevealMessage
	message.SendPrecompReveal(NextServer, msg)
}
