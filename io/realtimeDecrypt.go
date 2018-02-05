package io

import (
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/comms/mixserver/message"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/cryptops/realtime"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
)

// Blank struct for implementing services.BatchTransmission
type RealtimeDecryptHandler struct{}

// ReceptionHandler for RealtimeDecryptMessages
func (s ServerImpl) RealtimeDecrypt(input *pb.RealtimeDecryptMessage) {
	// Iterate through the Slots in the RealtimeDecryptMessage
	for i := 0; i < len(input.Slots); i++ {
		// Convert input message to equivalent SlotDecrypt
		in := input.Slots[i]
		var slot services.Slot = &realtime.SlotDecryptIn{
			Slot:                 in.Slot,
			SenderID:             in.SenderID,
			EncryptedMessage:     cyclic.NewIntFromBytes(in.EncryptedMessage),
			EncryptedRecipientID: cyclic.NewIntFromBytes(in.EncryptedRecipientID),
			TransmissionKey:      cyclic.NewMaxInt(), // TODO populate this field
		}
		// Pass slot as input to Decrypt's channel
		s.GetChannel(input.RoundID, globals.REAL_DECRYPT) <- &slot
	}
}

// TransmissionHandler for RealtimeDecryptMessages
func (h RealtimeDecryptHandler) Handler(
	roundId string, batchSize uint64, slots []*services.Slot) {
	// Create the RealtimeDecryptMessage
	msg := &pb.RealtimeDecryptMessage{
		RoundID: roundId,
		Slots:   make([]*pb.RealtimeDecryptSlot, batchSize),
	}

	// Iterate over the output channel
	for i := uint64(0); i < batchSize; i++ {
		// Type assert Slot to SlotDecrypt
		out := (*slots[i]).(*realtime.SlotDecryptOut)
		// Convert to RealtimeDecryptSlot
		msgSlot := &pb.RealtimeDecryptSlot{
			Slot:                 out.Slot,
			SenderID:             out.SenderID,
			EncryptedMessage:     out.EncryptedMessage.Bytes(),
			EncryptedRecipientID: out.EncryptedRecipientID.Bytes(),
		}

		// Append the RealtimeDecryptSlot to the RealtimeDecryptMessage
		msg.Slots[i] = msgSlot
	}
	// Send the completed RealtimeDecryptMessage
	message.SendRealtimeDecrypt(NextServer, msg)
}
