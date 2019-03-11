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
type PrecompDecryptHandler struct{}

// ReceptionHandler for PrecompDecryptMessages
func PrecompDecrypt(input *pb.PrecompDecryptMessage) {
	startTime := time.Now()
	jww.INFO.Printf("Starting PrecompDecrypt(RoundId: %s, Phase: %s) at %s",
		input.RoundID, globals.Phase(input.LastOp).String(),
		startTime.Format(time.RFC3339))

	// Get the input channel for the cryptop
	chIn := GetChannel(input.RoundID, globals.PRECOMP_DECRYPT)

	// Store when the operation started
	globals.GlobalRoundMap.GetRound(input.RoundID).CryptopStartTimes[globals.
		PRECOMP_DECRYPT] = startTime

	// Iterate through the Slots in the PrecompDecryptMessage
	for i := 0; i < len(input.Slots); i++ {
		// Convert input message to equivalent SlotDecrypt
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
		// Pass slot as input to Decrypt's channel
		chIn <- &slot
	}

	close(chIn)

	endTime := time.Now()
	jww.INFO.Printf("Finished PrecompDecrypt(RoundId: %s, Phase: %s) in %d ms",
		input.RoundID, globals.Phase(input.LastOp).String(),
		(endTime.Sub(startTime))/time.Millisecond)
}

// Transition to PrecompPermute phase on the last node
func precompDecryptLastNode(roundId string, batchSize uint64,
	input *pb.PrecompDecryptMessage) {

	startTime := time.Now()
	jww.INFO.Printf("[Last Node] Initializing PrecompPermute(RoundId: %s, "+
		"Phase: %s) at %s",
		input.RoundID, globals.Phase(input.LastOp).String(),
		startTime.Format(time.RFC3339))

	// TODO: record the start precomp permute time for this round here,
	//       and print the time it took for the Decrypt phase to complete.

	// Store when the operation started
	globals.GlobalRoundMap.GetRound(input.RoundID).CryptopStartTimes[globals.
		PRECOMP_DECRYPT] = startTime

	// Create the PrecompPermuteMessage
	msg := &pb.PrecompPermuteMessage{
		RoundID: roundId,
		LastOp:  int32(globals.PRECOMP_DECRYPT),
		Slots:   make([]*pb.PrecompPermuteSlot, batchSize),
	}

	// Iterate over the input slots
	for i := range input.Slots {
		out := input.Slots[i]
		// Convert to PrecompPermuteSlot
		msgSlot := &pb.PrecompPermuteSlot{
			Slot:                            out.Slot,
			EncryptedMessageKeys:            out.EncryptedMessageKeys,
			EncryptedAssociatedDataKeys:     out.EncryptedAssociatedDataKeys,
			PartialMessageCypherText:        out.PartialMessageCypherText,
			PartialAssociatedDataCypherText: out.PartialAssociatedDataCypherText,
		}

		// Append the PrecompPermuteSlot to the PrecompPermuteMessage
		msg.Slots[i] = msgSlot
	}

	// Send the first PrecompPermute Message
	sendTime := time.Now()
	jww.INFO.Printf("[Last Node] Sending PrecompPermute Messages to %v at %s",
		NextServer, sendTime.Format(time.RFC3339))
	node.SendPrecompPermute(NextServer, msg)

	endTime := time.Now()
	jww.INFO.Printf("[Last Node] Finished Initializing "+
		"PrecompPermute(RoundId: %s, Phase: %s) in %d ms",
		input.RoundID, globals.Phase(input.LastOp).String(),
		(endTime.Sub(startTime))/time.Millisecond)
}

// TransmissionHandler for PrecompDecryptMessages
func (h PrecompDecryptHandler) Handler(
	roundId string, batchSize uint64, slots []*services.Slot) {
	startTime := time.Now()

	elapsed := startTime.Sub(globals.GlobalRoundMap.GetRound(roundId).
		CryptopStartTimes[globals.PRECOMP_DECRYPT])

	jww.DEBUG.Printf("PrecompDecrypt Crypto took %v ms for "+
		"RoundId %s", elapsed, roundId)

	jww.INFO.Printf("Starting PrecompDecrypt.Handler(RoundId: %s) at %s",
		roundId, startTime.Format(time.RFC3339))

	// Create the PrecompDecryptMessage
	msg := &pb.PrecompDecryptMessage{
		RoundID: roundId,
		LastOp:  int32(globals.PRECOMP_DECRYPT),
		Slots:   make([]*pb.PrecompDecryptSlot, batchSize),
	}

	// Iterate over the output channel
	for i := uint64(0); i < batchSize; i++ {
		// Type assert Slot to SlotDecrypt
		out := (*slots[i]).(*precomputation.PrecomputationSlot)
		// Convert to PrecompDecryptSlot
		msgSlot := &pb.PrecompDecryptSlot{
			Slot:                            out.Slot,
			EncryptedMessageKeys:            out.MessageCypher.Bytes(),
			EncryptedAssociatedDataKeys:     out.AssociatedDataCypher.Bytes(),
			PartialMessageCypherText:        out.MessagePrecomputation.Bytes(),
			PartialAssociatedDataCypherText: out.AssociatedDataPrecomputation.Bytes(),
		}

		// Append the PrecompDecryptSlot to the PrecompDecryptMessage
		msg.Slots[i] = msgSlot
	}

	// Advance internal state to PRECOMP_PERMUTE (the next phase)
	globals.GlobalRoundMap.SetPhase(roundId, globals.PRECOMP_PERMUTE)

	sendTime := time.Now()
	if id.IsLastNode {
		// Transition to PrecompPermute phase
		jww.INFO.Printf("Starting PrecompPermute Phase to %v at %s",
			NextServer, sendTime.Format(time.RFC3339))
		precompDecryptLastNode(roundId, batchSize, msg)
	} else {
		// Send the completed PrecompDecryptMessage
		jww.INFO.Printf("Sending PrecompDecrypt Message to %v at %s",
			NextServer, sendTime.Format(time.RFC3339))
		node.SendPrecompDecrypt(NextServer, msg)
	}

	endTime := time.Now()
	jww.INFO.Printf("Finished PrecompDecrypt.Handler(RoundId: %s) in %d ms",
		roundId, (endTime.Sub(startTime))/time.Millisecond)
}
