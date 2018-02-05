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
type RealtimeEncryptHandler struct{}

// ReceptionHandler for RealtimeEncryptMessages
func (s ServerImpl) RealtimeEncrypt(input *pb.RealtimeEncryptMessage) {
	// Iterate through the Slots in the RealtimeEncryptMessage
	for i := 0; i < len(input.Slots); i++ {
		// Convert input message to equivalent SlotEncrypt
		in := input.Slots[i]
		var slot services.Slot = &realtime.SlotEncryptIn{
			Slot:             in.Slot,
			RecipientID:      in.RecipientID,
			EncryptedMessage: cyclic.NewIntFromBytes(in.EncryptedMessage),
			ReceptionKey:     cyclic.NewMaxInt(), // TODO populate this field
		}
		// Pass slot as input to Encrypt's channel
		s.GetChannel(input.RoundID, globals.REAL_ENCRYPT) <- &slot
	}
}

// TransmissionHandler for RealtimeEncryptMessages
func (h RealtimeEncryptHandler) Handler(
	roundId string, batchSize uint64, slots []*services.Slot) {
	// Create the RealtimeEncryptMessage
	msg := &pb.RealtimeEncryptMessage{
		RoundID: roundId,
		Slots:   make([]*pb.RealtimeEncryptSlot, batchSize),
	}

	// Iterate over the output channel
	for i := uint64(0); i < batchSize; i++ {
		// Type assert Slot to SlotEncrypt
		out := (*slots[i]).(*realtime.SlotEncryptOut)
		// Convert to RealtimeEncryptSlot
		msgSlot := &pb.RealtimeEncryptSlot{
			Slot:             out.Slot,
			RecipientID:      out.RecipientID,
			EncryptedMessage: out.EncryptedMessage.Bytes(),
		}

		// Append the RealtimeEncryptSlot to the RealtimeEncryptMessage
		msg.Slots[i] = msgSlot
	}
	// Send the completed RealtimeEncryptMessage
	message.SendRealtimeEncrypt(NextServer, msg)
}
