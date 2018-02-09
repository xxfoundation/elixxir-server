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
type PrecompEncryptHandler struct{}

// ReceptionHandler for PrecompEncryptMessages
func (s ServerImpl) PrecompEncrypt(input *pb.PrecompEncryptMessage) {
	chIn := s.GetChannel(input.RoundID, globals.PRECOMP_ENCRYPT)
	// Iterate through the Slots in the PrecompEncryptMessage
	for i := 0; i < len(input.Slots); i++ {
		// Convert input message to equivalent SlotEncrypt
		in := input.Slots[i]
		var slot services.Slot = &precomputation.SlotEncrypt{
			Slot: in.Slot,
			EncryptedMessageKeys: cyclic.NewIntFromBytes(
				in.EncryptedMessageKeys),
			PartialMessageCypherText: cyclic.NewIntFromBytes(
				in.PartialMessageCypherText),
		}
		// Pass slot as input to Encrypt's channel
		chIn <- &slot
	}
}

// Transition to PrecompReveal phase on the last node
func precompEncryptLastNode(roundId string, batchSize uint64,
	input *pb.PrecompEncryptMessage) {
	jww.INFO.Println("Beginning PrecompReveal Phase...")
	// Create the PrecompRevealMessage for sending
	msg := &pb.PrecompRevealMessage{
		RoundID: roundId,
		Slots:   make([]*pb.PrecompRevealSlot, batchSize),
	}

	round := globals.GlobalRoundMap.GetRound(roundId)
	// Iterate over the input slots
	for i := uint64(0); i < batchSize; i++ {
		out := input.Slots[i]
		// Convert to PrecompRevealSlot
		msgSlot := &pb.PrecompRevealSlot{
			Slot: out.Slot,
			PartialMessageCypherText:   out.PartialMessageCypherText,
			PartialRecipientCypherText: round.LastNode.RecipientCypherText[i].Bytes(),
		}

		// Save the Message Precomputation
		round.LastNode.EncryptedMessagePrecomputation[i].SetBytes(
			out.EncryptedMessageKeys)

		// Append the PrecompRevealSlot to the PrecompRevealMessage
		msg.Slots[i] = msgSlot
	}

	// Send the first PrecompRevealMessage
	jww.INFO.Printf("Sending PrecompReveal Message to %v...", NextServer)
	message.SendPrecompReveal(NextServer, msg)
}

// TransmissionHandler for PrecompEncryptMessages
func (h PrecompEncryptHandler) Handler(
	roundId string, batchSize uint64, slots []*services.Slot) {
	// Create the PrecompEncryptMessage
	msg := &pb.PrecompEncryptMessage{
		RoundID: roundId,
		Slots:   make([]*pb.PrecompEncryptSlot, batchSize),
	}

	// Iterate over the output channel
	for i := uint64(0); i < batchSize; i++ {
		// Type assert Slot to SlotEncrypt
		out := (*slots[i]).(*precomputation.SlotEncrypt)
		// Convert to PrecompEncryptSlot
		msgSlot := &pb.PrecompEncryptSlot{
			Slot:                     out.Slot,
			EncryptedMessageKeys:     out.EncryptedMessageKeys.Bytes(),
			PartialMessageCypherText: out.PartialMessageCypherText.Bytes(),
		}

		// Put it into the slice
		msg.Slots[i] = msgSlot
	}

	if IsLastNode {
		// Transition to PrecompReveal phase
		precompEncryptLastNode(roundId, batchSize, msg)
	} else {
		// Send the completed PrecompEncryptMessage
		jww.INFO.Printf("Sending PrecompEncrypt Message to %v...", NextServer)
		message.SendPrecompEncrypt(NextServer, msg)
	}
}
