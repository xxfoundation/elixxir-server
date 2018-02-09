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
type PrecompDecryptHandler struct{}

// ReceptionHandler for PrecompDecryptMessages
func (s ServerImpl) PrecompDecrypt(input *pb.PrecompDecryptMessage) {
	jww.INFO.Printf("Received PrecompDecrypt Message %v...", input.RoundID)
	// Get the input channel for the cryptop
	chIn := s.GetChannel(input.RoundID, globals.PRECOMP_DECRYPT)
	// Iterate through the Slots in the PrecompDecryptMessage
	for i := 0; i < len(input.Slots); i++ {
		// Convert input message to equivalent SlotDecrypt
		in := input.Slots[i]
		var slot services.Slot = &precomputation.SlotDecrypt{
			Slot:                         in.Slot,
			EncryptedMessageKeys:         cyclic.NewIntFromBytes(in.EncryptedMessageKeys),
			EncryptedRecipientIDKeys:     cyclic.NewIntFromBytes(in.EncryptedRecipientIDKeys),
			PartialMessageCypherText:     cyclic.NewIntFromBytes(in.PartialMessageCypherText),
			PartialRecipientIDCypherText: cyclic.NewIntFromBytes(in.PartialRecipientIDCypherText),
		}
		// Pass slot as input to Decrypt's channel
		chIn <- &slot
	}
}

// Convert Decrypt output slot to Permute input slot
func decryptPermuteTranslate(roundId string, batchSize uint64, decryptSlots []*services.Slot) {
	jww.INFO.Println("Beginning PrecompPermute Phase...")
	permuteSlots := make([]*services.Slot, len(decryptSlots))
	for i := range decryptSlots {
		is := precomputation.SlotPermute(*((*decryptSlots[i]).(*precomputation.SlotDecrypt)))
		os := services.Slot(&is)
		permuteSlots[i] = &os
	}
	PrecompPermuteHandler{}.Handler(roundId, batchSize, permuteSlots)
}

// TransmissionHandler for PrecompDecryptMessages
func (h PrecompDecryptHandler) Handler(
	roundId string, batchSize uint64, slots []*services.Slot) {
	// Create the PrecompDecryptMessage
	msg := &pb.PrecompDecryptMessage{
		RoundID: roundId,
		Slots:   make([]*pb.PrecompDecryptSlot, batchSize),
	}

	// Iterate over the output channel
	for i := uint64(0); i < batchSize; i++ {
		// Type assert Slot to SlotDecrypt
		out := (*slots[i]).(*precomputation.SlotDecrypt)
		// Convert to PrecompDecryptSlot
		msgSlot := &pb.PrecompDecryptSlot{
			Slot:                         out.Slot,
			EncryptedMessageKeys:         out.EncryptedMessageKeys.Bytes(),
			EncryptedRecipientIDKeys:     out.EncryptedRecipientIDKeys.Bytes(),
			PartialMessageCypherText:     out.PartialMessageCypherText.Bytes(),
			PartialRecipientIDCypherText: out.PartialRecipientIDCypherText.Bytes(),
		}

		// Append the PrecompDecryptSlot to the PrecompDecryptMessage
		msg.Slots[i] = msgSlot
	}
	// Send the completed PrecompDecryptMessage
	jww.INFO.Printf("Sending PrecompDecrypt Message to %v...", NextServer)
	if IsLastNode {
		decryptPermuteTranslate(roundId, batchSize, slots)
	} else {
		message.SendPrecompDecrypt(NextServer, msg)
	}
}
