// Copyright © 2018 Privategrity Corporation
//
// All rights reserved.
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
	jww.DEBUG.Printf("Received PrecompDecrypt Message %v from phase %s...",
		input.RoundID, globals.Phase(input.LastOp).String())
	// Get the input channel for the cryptop
	chIn := s.GetChannel(input.RoundID, globals.PRECOMP_DECRYPT)
	// Iterate through the Slots in the PrecompDecryptMessage
	for i := 0; i < len(input.Slots); i++ {
		// Convert input message to equivalent SlotDecrypt
		in := input.Slots[i]
		var slot services.Slot = &precomputation.PrecomputationSlot{
			Slot: in.Slot,
			MessageCypher: cyclic.NewIntFromBytes(
				in.EncryptedMessageKeys),
			RecipientIDCypher: cyclic.NewIntFromBytes(
				in.EncryptedRecipientIDKeys),
			MessagePrecomputation: cyclic.NewIntFromBytes(
				in.PartialMessageCypherText),
			RecipientIDPrecomputation: cyclic.NewIntFromBytes(
				in.PartialRecipientIDCypherText),
		}
		// Pass slot as input to Decrypt's channel
		chIn <- &slot
	}
}

// Transition to PrecompPermute phase on the last node
func precompDecryptLastNode(roundId string, batchSize uint64,
	input *pb.PrecompDecryptMessage) {
	jww.INFO.Println("Beginning PrecompPermute Phase...")
	// Create the PrecompPermuteMessage
	msg := &pb.PrecompPermuteMessage{
		RoundID: roundId,
		LastOp:  int32(globals.PRECOMP_DECRYPT),
		Slots:   make([]*pb.PrecompPermuteSlot, batchSize),
	}

	// Iterate over the input slots
	for i := range input.Slots {
		out := input.Slots[i]
		// Convert to PrecompPermuteSlot
		msgSlot := &pb.PrecompPermuteSlot{
			Slot:                         out.Slot,
			EncryptedMessageKeys:         out.EncryptedMessageKeys,
			EncryptedRecipientIDKeys:     out.EncryptedRecipientIDKeys,
			PartialMessageCypherText:     out.PartialMessageCypherText,
			PartialRecipientIDCypherText: out.PartialRecipientIDCypherText,
		}

		// Append the PrecompPermuteSlot to the PrecompPermuteMessage
		msg.Slots[i] = msgSlot
	}

	// Send the first PrecompPermute Message
	jww.DEBUG.Printf("Sending PrecompPermute Message to %v...", NextServer)
	message.SendPrecompPermute(NextServer, msg)
}

// TransmissionHandler for PrecompDecryptMessages
func (h PrecompDecryptHandler) Handler(
	roundId string, batchSize uint64, slots []*services.Slot) {
	// Create the PrecompDecryptMessage
	msg := &pb.PrecompDecryptMessage{
		RoundID: roundId,
		LastOp:  int32(globals.PRECOMP_DECRYPT),
		Slots:   make([]*pb.PrecompDecryptSlot, batchSize),
	}

	// Iterate over the output channel
	for i := uint64(0); i < batchSize; i++ {
		// Type assert Slot to SlotDecrypt
		out := (*slots[i]).(*precomputation.PrecomputationSlot)
		// Convert to PrecompDecryptSlot
		msgSlot := &pb.PrecompDecryptSlot{
			Slot:                         out.Slot,
			EncryptedMessageKeys:         out.MessageCypher.Bytes(),
			EncryptedRecipientIDKeys:     out.RecipientIDCypher.Bytes(),
			PartialMessageCypherText:     out.MessagePrecomputation.Bytes(),
			PartialRecipientIDCypherText: out.RecipientIDPrecomputation.Bytes(),
		}

		// Append the PrecompDecryptSlot to the PrecompDecryptMessage
		msg.Slots[i] = msgSlot
	}

	if IsLastNode {
		// Transition to PrecompPermute phase
		precompDecryptLastNode(roundId, batchSize, msg)
	} else {
		// Send the completed PrecompDecryptMessage
		jww.DEBUG.Printf("Sending PrecompDecrypt Message to %v...", NextServer)
		message.SendPrecompDecrypt(NextServer, msg)
	}
}
