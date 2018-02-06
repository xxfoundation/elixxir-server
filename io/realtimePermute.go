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
type RealtimePermuteHandler struct{}

// ReceptionHandler for RealtimePermuteMessages
func (s ServerImpl) RealtimePermute(input *pb.RealtimePermuteMessage) {
	// Get the input channel for the cryptop
	chIn := s.GetChannel(input.RoundID, globals.REAL_PERMUTE)
	// Iterate through the Slots in the RealtimePermuteMessage
	for i := 0; i < len(input.Slots); i++ {
		// Convert input message to equivalent SlotPermute
		in := input.Slots[i]
		var slot services.Slot = &realtime.SlotPermute{
			Slot: in.Slot,
			EncryptedMessage: cyclic.NewIntFromBytes(
				in.EncryptedMessage),
			EncryptedRecipientID: cyclic.NewIntFromBytes(
				in.EncryptedRecipientID),
		}
		// Pass slot as input to Permute's channel
		chIn <- &slot
	}
}

// TransmissionHandler for RealtimePermuteMessages
func (h RealtimePermuteHandler) Handler(
	roundId string, batchSize uint64, slots []*services.Slot) {
	// Create the RealtimePermuteMessage for sending
	msg := &pb.RealtimePermuteMessage{
		RoundID: roundId,
		Slots:   make([]*pb.RealtimePermuteSlot, batchSize),
	}

	// Iterate over the output channel
	for i := uint64(0); i < batchSize; i++ {
		// Type assert Slot to SlotPermute
		out := (*slots[i]).(*realtime.SlotPermute)
		// Convert to RealtimePermuteSlot
		msgSlot := &pb.RealtimePermuteSlot{
			Slot:                 out.Slot,
			EncryptedMessage:     out.EncryptedMessage.Bytes(),
			EncryptedRecipientID: out.EncryptedRecipientID.Bytes(),
		}

		// Append the RealtimePermuteSlot to the RealtimePermuteMessage
		msg.Slots[i] = msgSlot
	}
	// Send the completed RealtimePermuteMessage
	message.SendRealtimePermute(NextServer, msg)
}
