package io

import (
	jww "github.com/spf13/jwalterweatherman"
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
	jww.DEBUG.Printf("Received RealtimeDecrypt Message %v...", input.RoundID)
	// Get the input channel for the cryptop
	chIn := s.GetChannel(input.RoundID, globals.REAL_DECRYPT)
	// Iterate through the Slots in the RealtimeDecryptMessage
	for i := 0; i < len(input.Slots); i++ {
		// Convert input message to equivalent SlotDecrypt
		in := input.Slots[i]
		var slot services.Slot = &realtime.SlotDecryptIn{
			Slot:                 in.Slot,
			SenderID:             in.SenderID,
			EncryptedMessage:     cyclic.NewIntFromBytes(in.EncryptedMessage),
			EncryptedRecipientID: cyclic.NewIntFromBytes(in.EncryptedRecipientID),
			TransmissionKey:      cyclic.NewInt(1),
		}
		// Pass slot as input to Decrypt's channel
		chIn <- &slot
	}
}

// Transition to RealtimePermute phase on the last node
func realtimeDecryptLastNode(roundId string, batchSize uint64,
	input *pb.RealtimeDecryptMessage) {
	jww.INFO.Println("Beginning RealtimePermute Phase...")
	// Create the RealtimePermuteMessage
	msg := &pb.RealtimePermuteMessage{
		RoundID: roundId,
		Slots:   make([]*pb.RealtimePermuteSlot, batchSize),
	}

	// Iterate over the input slots
	for i := range input.Slots {
		out := input.Slots[i]
		// Convert to RealtimePermuteSlot
		msgSlot := &pb.RealtimePermuteSlot{
			Slot:                 out.Slot,
			EncryptedMessage:     out.EncryptedMessage,
			EncryptedRecipientID: out.EncryptedRecipientID,
		}

		// Append the RealtimePermuteSlot to the RealtimePermuteMessage
		msg.Slots[i] = msgSlot
	}

	// Send the first RealtimePermute Message
	jww.DEBUG.Printf("Sending RealtimePermute Message to %v...", NextServer)
	message.SendRealtimePermute(NextServer, msg)
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

	if IsLastNode {
		// Transition to RealtimePermute phase
		realtimeDecryptLastNode(roundId, batchSize, msg)
	} else {
		// Send the completed RealtimeDecryptMessage
		jww.DEBUG.Printf("Sending RealtimeDecrypt Message to %v...", NextServer)
		message.SendRealtimeDecrypt(NextServer, msg)
	}
}

// Kickoff for RealtimeDecryptMessages
// TODO Remove this duplication
func KickoffDecryptHandler(roundId string, batchSize uint64, slots []*services.Slot) {
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
	jww.DEBUG.Printf("Sending RealtimeDecrypt Message to %v...", NextServer)
	message.SendRealtimeDecrypt(NextServer, msg)
}
