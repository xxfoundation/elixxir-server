////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

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
	jww.DEBUG.Printf("Received RealtimeDecrypt Message %v from phase %s...",
		input.RoundID, globals.Phase(input.LastOp).String())
	// Get the input channel for the cryptop
	chIn := s.GetChannel(input.RoundID, globals.REAL_DECRYPT)
	// Iterate through the Slots in the RealtimeDecryptMessage
	for i := 0; i < len(input.Slots); i++ {
		// Convert input message to equivalent SlotDecrypt
		in := input.Slots[i]
		var slot services.Slot = &realtime.RealtimeSlot{
			Slot:               in.Slot,
			CurrentID:          in.SenderID,
			Message:            cyclic.NewIntFromBytes(in.EncryptedMessage),
			EncryptedRecipient: cyclic.NewIntFromBytes(in.EncryptedRecipientID),
			CurrentKey:         cyclic.NewInt(1),
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
		LastOp:  int32(globals.REAL_DECRYPT),
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
		LastOp:  int32(globals.REAL_DECRYPT),
		Slots:   make([]*pb.RealtimeDecryptSlot, batchSize),
	}

	// Iterate over the output channel
	for i := uint64(0); i < batchSize; i++ {
		// Type assert Slot to SlotDecrypt
		out := (*slots[i]).(*realtime.RealtimeSlot)
		// Convert to RealtimeDecryptSlot
		msgSlot := &pb.RealtimeDecryptSlot{
			Slot:                 out.Slot,
			SenderID:             out.CurrentID,
			EncryptedMessage:     out.Message.Bytes(),
			EncryptedRecipientID: out.EncryptedRecipient.Bytes(),
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
func KickoffDecryptHandler(roundId string, batchSize uint64,
	slots []*realtime.RealtimeSlot) {
	// Create the RealtimeDecryptMessage
	msg := &pb.RealtimeDecryptMessage{
		RoundID: roundId,
		LastOp:  int32(globals.WAIT),
		Slots:   make([]*pb.RealtimeDecryptSlot, batchSize),
	}

	// Iterate over the output channel
	for i := uint64(0); i < batchSize; i++ {
		// Type assert Slot to SlotDecrypt
		out := slots[i]
		// Convert to RealtimeDecryptSlot
		msgSlot := &pb.RealtimeDecryptSlot{
			Slot:                 out.Slot,
			SenderID:             out.CurrentID,
			EncryptedMessage:     out.Message.Bytes(),
			EncryptedRecipientID: out.EncryptedRecipient.Bytes(),
		}

		// Append the RealtimeDecryptSlot to the RealtimeDecryptMessage
		msg.Slots[i] = msgSlot
	}
	// Send the completed RealtimeDecryptMessage
	jww.DEBUG.Printf("Sending RealtimeDecrypt Message to %v...", NextServer)
	message.SendRealtimeDecrypt(NextServer, msg)
}
