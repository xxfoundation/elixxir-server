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
			PartialRecipientCypherText: cyclic.NewIntFromBytes(
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
	for i := uint64(0); i < batchSize; i++ {
		out := input.Slots[i]
		// Convert to SlotStripIn
		var slot services.Slot = &precomputation.SlotStripIn{
			Slot: out.Slot,
			RoundMessagePrivateKey:   cyclic.NewIntFromBytes(out.PartialMessageCypherText),
			RoundRecipientPrivateKey: cyclic.NewIntFromBytes(out.PartialRecipientCypherText),
		}
		// Pass slot as input to Strip's channel
		globals.GlobalRoundMap.GetRound(roundId).GetChannel(globals.PRECOMP_STRIP) <- &slot
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
			PartialMessageCypherText:   out.PartialMessageCypherText.Bytes(),
			PartialRecipientCypherText: out.PartialRecipientCypherText.Bytes(),
		}

		// Put it into the slice
		msg.Slots[i] = msgSlot
	}

	if IsLastNode {
		// Transition to PrecompStrip phase
		precompRevealLastNode(roundId, batchSize, msg)
	} else {
		// Send the completed PrecompRevealMessage
		jww.INFO.Printf("Sending PrecompReveal Message to %v...", NextServer)
		message.SendPrecompReveal(NextServer, msg)
	}
}
