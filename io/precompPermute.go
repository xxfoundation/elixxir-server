////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/privategrity/comms/node"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/cryptops/precomputation"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"

	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"time"
)

// Blank struct for implementing services.BatchTransmission
type PrecompPermuteHandler struct{}

// ReceptionHandler for PrecompPermuteMessages
func (s ServerImpl) PrecompPermute(input *pb.PrecompPermuteMessage) {
	startTime := time.Now()
	jww.INFO.Printf("Starting PrecompPermute(RoundId: %s, Phase: %s) at %s",
		input.RoundID, globals.Phase(input.LastOp).String(),
		startTime.Format(time.RFC3339))

	// Get the input channel for the cryptop
	defer recoverSetPhasePanic(input.RoundID)
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

	close(chIn)

	endTime := time.Now()
	jww.INFO.Printf("Finished PrecompPermute(RoundId: %s, Phase: %s) in %d ms",
		input.RoundID, globals.Phase(input.LastOp).String(),
		(endTime.Sub(startTime))/time.Millisecond)
}

// Save the recipient cyphertext and the encrypted recipient precomputation,
// Send the encrypted message keys and partial message cypher text to the first
// nodes Encrypt handler
// Transition to PrecompEncrypt phase on the last node
func precompPermuteLastNode(roundId string, batchSize uint64,
	input *pb.PrecompPermuteMessage) {

	startTime := time.Now()
	jww.INFO.Printf("[Last Node] Initializing PrecompEncrypt(RoundId: %s, "+
		"Phase: %s) at %s",
		input.RoundID, globals.Phase(input.LastOp).String(),
		startTime.Format(time.RFC3339))

	// TODO: record the start precomp permute time for this round here,
	//       and print the time it took for the Decrypt phase to complete.

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
	sendTime := time.Now()
	jww.INFO.Printf("[Last Node] Sending PrecompEncrypt Messages to %v at %s",
		NextServer, sendTime.Format(time.RFC3339))
	node.SendPrecompEncrypt(NextServer, msg)

	endTime := time.Now()
	jww.INFO.Printf("[Last Node] Finished Initializing "+
		"PrecompEncrypt(RoundId: %s, Phase: %s) in %d ms",
		input.RoundID, globals.Phase(input.LastOp).String(),
		(endTime.Sub(startTime))/time.Millisecond)
}

// TransmissionHandler for PrecompPermuteMessages
func (h PrecompPermuteHandler) Handler(
	roundId string, batchSize uint64, slots []*services.Slot) {
	startTime := time.Now()
	jww.INFO.Printf("Starting PrecompPermute.Handler(RoundId: %s) at %s",
		roundId, startTime.Format(time.RFC3339))

	// Create the PrecompPermuteMessage for sending
	msg := &pb.PrecompPermuteMessage{
		RoundID: roundId,
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

	// Advance internal state to PRECOMP_PERMUTE (the next phase)
	defer recoverSetPhasePanic(roundId)
	globals.GlobalRoundMap.GetRound(roundId).SetPhase(globals.PRECOMP_ENCRYPT)

	sendTime := time.Now()
	if globals.IsLastNode {
		// Transition to PrecompEncrypt phase
		jww.INFO.Printf("Starting PrecompEncrypt Phase to %v at %s",
			NextServer, sendTime.Format(time.RFC3339))
		precompPermuteLastNode(roundId, batchSize, msg)
	} else {
		// Send the completed PrecompPermuteMessage
		jww.INFO.Printf("Sending PrecompPermute Message to %v at %s",
			NextServer, sendTime.Format(time.RFC3339))
		node.SendPrecompPermute(NextServer, msg)
	}

	endTime := time.Now()
	jww.INFO.Printf("Finished PrecompPermute.Handler(RoundId: %s) in %d ms",
		roundId, (endTime.Sub(startTime))/time.Millisecond)
}
