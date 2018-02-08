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
	// Send the completed PrecompEncryptMessage
	jww.INFO.Printf("Sending PrecompEncrypt Message to %v...", NextServer)
	message.SendPrecompEncrypt(NextServer, msg)
}
