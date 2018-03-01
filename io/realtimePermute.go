////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/comms/clusterclient"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/cryptops/realtime"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
)

// Blank struct for implementing services.BatchTransmission
type RealtimePermuteHandler struct{}

// ReceptionHandler for RealtimePermuteMessages
func (s ServerImpl) RealtimePermute(input *pb.RealtimePermuteMessage) {
	jww.DEBUG.Printf("Received RealtimePermute Message %v from phase %s...",
		input.RoundID, globals.Phase(input.LastOp).String())
	// Get the input channel for the cryptop
	chIn := s.GetChannel(input.RoundID, globals.REAL_PERMUTE)
	// Iterate through the Slots in the RealtimePermuteMessage
	for i := 0; i < len(input.Slots); i++ {
		// Convert input message to equivalent RealtimeSlot
		in := input.Slots[i]
		var slot services.Slot = &realtime.RealtimeSlot{
			Slot: in.Slot,
			Message: cyclic.NewIntFromBytes(
				in.EncryptedMessage),
			EncryptedRecipient: cyclic.NewIntFromBytes(
				in.EncryptedRecipientID),
		}
		// Pass slot as input to Permute's channel
		chIn <- &slot
	}
}

// Transition to RealtimeIdentify phase on the last node
func realtimePermuteLastNode(roundId string, batchSize uint64,
	input *pb.RealtimePermuteMessage) {
	jww.INFO.Println("Beginning RealtimeIdentify Phase...")
	// Get round and channel
	round := globals.GlobalRoundMap.GetRound(roundId)
	identifyChannel := round.GetChannel(globals.REAL_IDENTIFY)
	// Create the RealtimeSlot for sending into RealtimeIdentify
	for i := uint64(0); i < batchSize; i++ {
		out := input.Slots[i]
		// Convert to RealtimeSlot
		var slot services.Slot = &realtime.RealtimeSlot{
			Slot:               out.Slot,
			EncryptedRecipient: cyclic.NewIntFromBytes(out.EncryptedRecipientID),
		}
		// Save EncryptedMessages for the Identify->Encrypt transition
		round.LastNode.EncryptedMessage[i] = cyclic.NewIntFromBytes(out.EncryptedMessage)
		// Pass slot as input to Identify's channel
		identifyChannel <- &slot
	}
}

// TransmissionHandler for RealtimePermuteMessages
func (h RealtimePermuteHandler) Handler(
	roundId string, batchSize uint64, slots []*services.Slot) {
	// Create the RealtimePermuteMessage for sending
	msg := &pb.RealtimePermuteMessage{
		RoundID: roundId,
		LastOp:  int32(globals.REAL_PERMUTE),
		Slots:   make([]*pb.RealtimePermuteSlot, batchSize),
	}

	// Iterate over the output channel
	for i := uint64(0); i < batchSize; i++ {
		// Type assert Slot to RealtimeSlot
		out := (*slots[i]).(*realtime.RealtimeSlot)
		// Convert to RealtimePermuteSlot
		msgSlot := &pb.RealtimePermuteSlot{
			Slot:                 out.Slot,
			EncryptedMessage:     out.Message.Bytes(),
			EncryptedRecipientID: out.EncryptedRecipient.Bytes(),
		}

		// Append the RealtimePermuteSlot to the RealtimePermuteMessage
		msg.Slots[i] = msgSlot
	}

	if IsLastNode {
		// Transition to RealtimeIdentify phase
		realtimePermuteLastNode(roundId, batchSize, msg)
	} else {
		// Send the completed RealtimePermuteMessage
		jww.DEBUG.Printf("Sending RealtimePermute Message to %v...", NextServer)
		clusterclient.SendRealtimePermute(NextServer, msg)
	}
}
