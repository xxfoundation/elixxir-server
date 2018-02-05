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
type PrecompPermuteHandler struct{}

// ReceptionHandler for PrecompPermuteMessages
func (s ServerImpl) PrecompPermute(input *pb.PrecompPermuteMessage) {
	// Iterate through the Slots in the PrecompPermuteMessage
	for i := 0; i < len(input.Slots); i++ {
		// Convert input message to equivalent SlotPermute
		in := input.Slots[i]
		var slot services.Slot = &precomputation.SlotPermute{
			Slot: in.Slot,
			EncryptedMessageKeys: cyclic.NewIntFromBytes(
				in.EncryptedMessageKeys),
			EncryptedRecipientIDKeys: cyclic.NewIntFromBytes(
				in.EncryptedRecipientIDKeys),
			PartialMessageCypherText: cyclic.NewIntFromBytes(
				in.PartialMessageCypherText),
			PartialRecipientIDCypherText: cyclic.NewIntFromBytes(
				in.PartialRecipientIDCypherText),
		}
		// Pass slot as input to Permute's channel
		s.GetChannel(input.RoundID, globals.PRECOMP_PERMUTE) <- &slot
	}
}

// TransmissionHandler for PrecompPermuteMessages
func (h PrecompPermuteHandler) Handler(
	roundID string, batchSize uint64, slots []*services.Slot) {
	// Create the PrecompPermuteMessage for sending
	msg := &pb.PrecompPermuteMessage{
		RoundID: roundID,
		Slots:   make([]*pb.PrecompPermuteSlot, batchSize),
	}

	// Iterate over the output channel
	for i := uint64(0); i < batchSize; i++ {
		// Type assert Slot to SlotPermute
		out := (*slots[i]).(*precomputation.SlotPermute)
		// Convert to PrecompPermuteSlot
		msgSlot := &pb.PrecompPermuteSlot{
			Slot:                         out.Slot,
			EncryptedMessageKeys:         out.EncryptedMessageKeys.Bytes(),
			EncryptedRecipientIDKeys:     out.EncryptedRecipientIDKeys.Bytes(),
			PartialMessageCypherText:     out.PartialMessageCypherText.Bytes(),
			PartialRecipientIDCypherText: out.PartialRecipientIDCypherText.Bytes(),
		}

		// Append the PrecompPermuteSlot to the PrecompPermuteMessage
		msg.Slots = append(msg.Slots, msgSlot)
	}
	// Send the completed PrecompPermuteMessage
	message.SendPrecompPermute(NextServer, msg)
}
