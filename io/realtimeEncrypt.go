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
type RealtimeEncryptHandler struct{}

// ReceptionHandler for RealtimeEncryptMessages
func (s ServerImpl) RealtimeEncrypt(input *pb.RealtimeEncryptMessage) {
	jww.DEBUG.Printf("Received RealtimeEncrypt Message %v from phase %s...",
		input.RoundID, globals.Phase(input.LastOp).String())
	// Get the input channel for the cryptop
	chIn := s.GetChannel(input.RoundID, globals.REAL_ENCRYPT)
	// Iterate through the Slots in the RealtimeEncryptMessage
	for i := 0; i < len(input.Slots); i++ {
		// Convert input message to equivalent SlotEncrypt
		in := input.Slots[i]
		var slot services.Slot = &realtime.SlotEncryptIn{
			Slot:             in.Slot,
			RecipientID:      in.RecipientID,
			EncryptedMessage: cyclic.NewIntFromBytes(in.EncryptedMessage),
			ReceptionKey:     cyclic.NewInt(1),
		}
		// Pass slot as input to Encrypt's channel
		chIn <- &slot
	}
}

// Transition to RealtimePeel phase on the last node
func realtimeEncryptLastNode(roundId string, batchSize uint64,
	input *pb.RealtimeEncryptMessage) {
	jww.INFO.Println("Beginning RealtimePeel Phase...")
	// Get round and channel
	round := globals.GlobalRoundMap.GetRound(roundId)
	peelChannel := round.GetChannel(globals.REAL_PEEL)
	// Create the SlotPeel for sending into RealtimePeel
	for i := uint64(0); i < batchSize; i++ {
		out := input.Slots[i]
		// Convert to SlotPeel
		var slot services.Slot = &realtime.SlotPeel{
			Slot:             out.Slot,
			RecipientID:      out.RecipientID,
			EncryptedMessage: cyclic.NewIntFromBytes(out.EncryptedMessage),
		}
		// Pass slot as input to Peel's channel
		peelChannel <- &slot
	}
}

// TransmissionHandler for RealtimeEncryptMessages
func (h RealtimeEncryptHandler) Handler(
	roundId string, batchSize uint64, slots []*services.Slot) {
	// Create the RealtimeEncryptMessage
	msg := &pb.RealtimeEncryptMessage{
		RoundID: roundId,
		LastOp:  int32(globals.REAL_ENCRYPT),
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

	if IsLastNode {
		// Transition to RealtimePeel phase
		realtimeEncryptLastNode(roundId, batchSize, msg)
	} else {
		// Send the completed RealtimeEncryptMessage
		jww.DEBUG.Printf("Sending RealtimeEncrypt Message to %v...", NextServer)
		message.SendRealtimeEncrypt(NextServer, msg)
	}
}
