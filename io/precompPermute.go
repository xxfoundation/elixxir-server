////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/cryptops/precomputation"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/services"

	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/id"
	"time"
)

// Blank struct for implementing services.BatchTransmission
type PrecompPermuteHandler struct{}

// ReceptionHandler for PrecompPermuteMessages
func PrecompPermute(input *pb.PrecompPermuteMessage) {
	startTime := time.Now()
	jww.INFO.Printf("Starting PrecompPermute(RoundId: %s, Phase: %s) at %s",
		input.RoundID, globals.Phase(input.LastOp).String(),
		startTime.Format(time.RFC3339))

	// Get the input channel for the cryptop
	chIn := GetChannel(input.RoundID, globals.PRECOMP_PERMUTE)

	// Store when the operation started
	globals.GlobalRoundMap.GetRound(input.RoundID).CryptopStartTimes[globals.
		PRECOMP_PERMUTE] = startTime

	// Iterate through the Slots in the PrecompPermuteMessage
	for i := 0; i < len(input.Slots); i++ {
		// Convert input message to equivalent Slot
		in := input.Slots[i]
		var slot services.Slot = &precomputation.PrecomputationSlot{
			Slot: in.Slot,
			MessageCypher: cyclic.NewIntFromBytes(
				in.EncryptedMessageKeys),
			AssociatedDataCypher: cyclic.NewIntFromBytes(
				in.EncryptedAssociatedDataKeys),
			MessagePrecomputation: cyclic.NewIntFromBytes(
				in.PartialMessageCypherText),
			AssociatedDataPrecomputation: cyclic.NewIntFromBytes(
				in.PartialAssociatedDataCypherText),
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

// Save the AssociatedData cyphertext and the encrypted AssociatedData precomputation,
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
	if round == nil {
		jww.INFO.Printf("skipping round %s, because it's dead", roundId)
		return
	}

	// Iterate over the input slots
	for i := uint64(0); i < batchSize; i++ {
		out := input.Slots[i]
		// Convert to PrecompEncryptSlot
		msgSlot := &pb.PrecompEncryptSlot{
			Slot:                     out.Slot,
			EncryptedMessageKeys:     out.EncryptedMessageKeys,
			PartialMessageCypherText: out.PartialMessageCypherText,
		}

		// Save the AssociatedData CypherText and Precomputation
		round.LastNode.AssociatedDataCypherText[i].SetBytes(
			out.PartialAssociatedDataCypherText)
		round.LastNode.EncryptedAssociatedDataPrecomputation[i].SetBytes(
			out.EncryptedAssociatedDataKeys)

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

	elapsed := startTime.Sub(globals.GlobalRoundMap.GetRound(roundId).
		CryptopStartTimes[globals.PRECOMP_PERMUTE])

	jww.DEBUG.Printf("PrecompPermute Crypto took %v ms for "+
		"RoundId %s", elapsed, roundId)

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
		// Type assert Slot to Slot
		out := (*slots[i]).(*precomputation.PrecomputationSlot)
		// Convert to PrecompPermuteSlot
		msgSlot := &pb.PrecompPermuteSlot{
			Slot:                            out.Slot,
			EncryptedMessageKeys:            out.MessageCypher.Bytes(),
			EncryptedAssociatedDataKeys:     out.AssociatedDataCypher.Bytes(),
			PartialMessageCypherText:        out.MessagePrecomputation.Bytes(),
			PartialAssociatedDataCypherText: out.AssociatedDataPrecomputation.Bytes(),
		}

		// Append the PrecompPermuteSlot to the PrecompPermuteMessage
		msg.Slots[i] = msgSlot
	}

	// Advance internal state to PRECOMP_PERMUTE (the next phase)
	globals.GlobalRoundMap.SetPhase(roundId, globals.PRECOMP_ENCRYPT)

	sendTime := time.Now()
	if id.IsLastNode {
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
