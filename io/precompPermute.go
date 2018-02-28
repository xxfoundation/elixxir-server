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
	"gitlab.com/privategrity/server/cryptops/precomputation"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
)

// Blank struct for implementing services.BatchTransmission
type PrecompPermuteHandler struct{}

// ReceptionHandler for PrecompPermuteMessages
func (s ServerImpl) PrecompPermute(input *pb.PrecompPermuteMessage) {
	jww.DEBUG.Printf("Received PrecompPermute Message %v from phase %s...",
		input.RoundID, globals.Phase(input.LastOp).String())
	// Get the input channel for the cryptop
	chIn := s.GetChannel(input.RoundID, globals.PRECOMP_PERMUTE)
	// Iterate through the Slots in the PrecompPermuteMessage
	for i := 0; i < len(input.Slots); i++ {
		// Convert input message to equivalent RealtimeSlot
		in := input.Slots[i]
		var slot services.Slot = &precomputation.PrecomputationSlot{
			Slot: in.Slot,
			MessageCypher: cyclic.NewIntFromBytes(
				in.EncryptedMessageKeys),
			RecipientIDCypher: cyclic.NewIntFromBytes(
				in.EncryptedRecipientIDKeys),
			MessagePrecomputation: cyclic.NewIntFromBytes(
				in.PartialMessageCypherText),
			RecipientIDPrecomputation: cyclic.NewIntFromBytes(
				in.PartialRecipientIDCypherText),
		}
		// Pass slot as input to Permute's channel
		chIn <- &slot
	}
}

// Save the recipient cyphertext and the encrypted recipient precomputation,
// Send the encrypted message keys and partial message cypher text to the first
// nodes Encrypt handler
// Transition to PrecompEncrypt phase on the last node
func precompPermuteLastNode(roundId string, batchSize uint64,
	input *pb.PrecompPermuteMessage) {
	jww.INFO.Println("Beginning PrecompEncrypt Phase...")
	// Create the PrecompEncryptMessage for sending
	msg := &pb.PrecompEncryptMessage{
		RoundID: roundId,
		LastOp:  int32(globals.PRECOMP_PERMUTE),
		Slots:   make([]*pb.PrecompEncryptSlot, batchSize),
	}

	round := globals.GlobalRoundMap.GetRound(roundId)
	// Iterate over the input slots
	for i := uint64(0); i < batchSize; i++ {
		out := input.Slots[i]
		// Convert to PrecompEncryptSlot
		msgSlot := &pb.PrecompEncryptSlot{
			Slot:                     out.Slot,
			EncryptedMessageKeys:     out.EncryptedMessageKeys,
			PartialMessageCypherText: out.PartialMessageCypherText,
		}

		// Save the Recipient ID CypherText and Precomputation
		round.LastNode.RecipientCypherText[i].SetBytes(
			out.PartialRecipientIDCypherText)
		round.LastNode.EncryptedRecipientPrecomputation[i].SetBytes(
			out.EncryptedRecipientIDKeys)

		// Append the PrecompEncryptSlot to the PrecompEncryptMessage
		msg.Slots[i] = msgSlot
	}

	// Send the first PrecompEncrypt Message
	jww.DEBUG.Printf("Sending PrecompEncrypt Message to %v...", NextServer)
	clusterclient.SendPrecompEncrypt(NextServer, msg)
}

// TransmissionHandler for PrecompPermuteMessages
func (h PrecompPermuteHandler) Handler(
	roundID string, batchSize uint64, slots []*services.Slot) {
	// Create the PrecompPermuteMessage for sending
	msg := &pb.PrecompPermuteMessage{
		RoundID: roundID,
		LastOp:  int32(globals.PRECOMP_PERMUTE),
		Slots:   make([]*pb.PrecompPermuteSlot, batchSize),
	}

	// Iterate over the output channel
	for i := uint64(0); i < batchSize; i++ {
		// Type assert Slot to RealtimeSlot
		out := (*slots[i]).(*precomputation.PrecomputationSlot)
		// Convert to PrecompPermuteSlot
		msgSlot := &pb.PrecompPermuteSlot{
			Slot:                         out.Slot,
			EncryptedMessageKeys:         out.MessageCypher.Bytes(),
			EncryptedRecipientIDKeys:     out.RecipientIDCypher.Bytes(),
			PartialMessageCypherText:     out.MessagePrecomputation.Bytes(),
			PartialRecipientIDCypherText: out.RecipientIDPrecomputation.Bytes(),
		}

		// Append the PrecompPermuteSlot to the PrecompPermuteMessage
		msg.Slots[i] = msgSlot
	}

	if IsLastNode {
		// Transition to PrecompEncrypt phase
		precompPermuteLastNode(roundID, batchSize, msg)
	} else {
		// Send the completed PrecompPermuteMessage
		jww.DEBUG.Printf("Sending PrecompPermute Message to %v...", NextServer)
		clusterclient.SendPrecompPermute(NextServer, msg)
	}
}
