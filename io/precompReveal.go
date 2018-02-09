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
	chIn := s.GetChannel(input.RoundID, globals.PRECOMP_REVEAL)
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
		chIn <- &slot
	}
}

// Convert the Reveal message to a Strip message and send to the last node
func precompRevealLastNode(roundID string, batchSize uint64,
	slots []*pb.PrecompRevealSlot) {
	// Create the PrecompEncryptMessage for sending
	msg := &pb.PrecompStripMessage{
		RoundID: roundID,
		Slots:   make([]*pb.PrecompStripSlot, batchSize),
	}

	round := globals.GlobalRoundMap.GetRound(roundIdD)
	for i := uint64(0); i < batchSize; i++ {
		out := slots[i]
		// Convert to PrecompStripSlot
		msgSlot := &pb.PrecompStripSlot{
			Slot:                         out.Slot,
			RoundMessagePrivateKey:       out.PartialMessageCypherText,
			RoundRecipientPrivateKey:     out.PartialRecipientCypherText,
		}

		msg.Slots[i] = msgSlot
	}
	// Send the completed PrecompStripMessage
	jww.INFO.Printf("Sending PrecompStrip Message to %v...",
		Servers[len(Servers-1)])
	message.SendPrecompStrip(Servers[len(Servers-1)], msg)
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
	jww.INFO.Printf("Sending PrecompReveal Message to %v...", NextServer)
	message.SendPrecompReveal(NextServer, msg)
}
