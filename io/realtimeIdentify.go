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
type RealtimeIdentifyHandler struct{}

// ReceptionHandler for RealtimeIdentifyMessages
func (s ServerImpl) RealtimeIdentify(input *pb.RealtimeIdentifyMessage) {
	// Get the input channel for the cryptop
	chIn := s.GetChannel(input.RoundID, globals.REAL_IDENTIFY)
	// Iterate through the Slots in the RealtimeIdentifyMessage
	for i := 0; i < len(input.Slots); i++ {
		// Convert input message to equivalent SlotIdentify
		in := input.Slots[i]
		var slot services.Slot = &realtime.SlotIdentify{
			Slot:                 in.Slot,
			EncryptedRecipientID: cyclic.NewIntFromBytes(in.EncryptedRecipientID),
		}
		// Pass slot as input to Identify's channel
		chIn <- &slot
	}
}

// TransmissionHandler for RealtimeIdentifyMessages
func (h RealtimeIdentifyHandler) Handler(
	roundId string, batchSize uint64, slots []*services.Slot) {
	// Create the RealtimeIdentifyMessage
	msg := &pb.RealtimeIdentifyMessage{
		RoundID: roundId,
		Slots:   make([]*pb.RealtimeIdentifySlot, batchSize),
	}

	// Iterate over the output channel
	for i := uint64(0); i < batchSize; i++ {
		// Type assert Slot to SlotIdentify
		out := (*slots[i]).(*realtime.SlotIdentify)
		// Convert to RealtimeIdentifySlot
		msgSlot := &pb.RealtimeIdentifySlot{
			Slot:                 out.Slot,
			EncryptedRecipientID: out.EncryptedRecipientID.Bytes(),
		}

		// Append the RealtimeIdentifySlot to the RealtimeIdentifyMessage
		msg.Slots[i] = msgSlot
	}
	// Send the completed RealtimeIdentifyMessage
	message.SendRealtimeIdentify(NextServer, msg)
}
