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
type PrecompEncryptHandler struct{}

// ReceptionHandler for PrecompEncryptMessages
func (s ServerImpl) PrecompEncrypt(input *pb.PrecompEncryptMessage) {
	startTime := time.Now()
	jww.INFO.Printf("Starting PrecompEncrypt(RoundId: %s, Phase: %s) at %s",
		input.RoundID, globals.Phase(input.LastOp).String(),
		startTime.Format(time.RFC3339))

	chIn := s.GetChannel(input.RoundID, globals.PRECOMP_ENCRYPT)

	globals.GlobalRoundMap.GetRound(input.RoundID).CryptopStartTimes[globals.
		PRECOMP_ENCRYPT] = startTime

	jww.DEBUG.Printf("Received PrecompEncrypt Message %v from phase %s...",
		input.RoundID, globals.Phase(input.LastOp).String())
	// Iterate through the Slots in the PrecompEncryptMessage
	for i := 0; i < len(input.Slots); i++ {
		// Convert input message to equivalent SlotEncrypt
		in := input.Slots[i]
		var slot services.Slot = &precomputation.PrecomputationSlot{
			Slot:          in.Slot,
			MessageCypher: cyclic.NewIntFromBytes(in.EncryptedMessageKeys),
			MessagePrecomputation: cyclic.NewIntFromBytes(
				in.PartialMessageCypherText),
		}
		// Pass slot as input to Encrypt's channel
		chIn <- &slot
	}

	close(chIn)

	endTime := time.Now()
	jww.INFO.Printf("Finished PrecompEncrypt(RoundId: %s, Phase: %s) in %d ms",
		input.RoundID, globals.Phase(input.LastOp).String(),
		(endTime.Sub(startTime))/time.Millisecond)
}

// Transition to PrecompReveal phase on the last node
func precompEncryptLastNode(roundId string, batchSize uint64,
	input *pb.PrecompEncryptMessage) {

	startTime := time.Now()
	jww.INFO.Printf("[Last Node] Initializing PrecompReveal(RoundId: %s, "+
		"Phase: %s) at %s",
		input.RoundID, globals.Phase(input.LastOp).String(),
		startTime.Format(time.RFC3339))

	globals.GlobalRoundMap.GetRound(input.RoundID).CryptopStartTimes[globals.
		PRECOMP_ENCRYPT] = startTime

	// Create the PrecompRevealMessage for sending
	msg := &pb.PrecompRevealMessage{
		RoundID: roundId,
		LastOp:  int32(globals.PRECOMP_ENCRYPT),
		Slots:   make([]*pb.PrecompRevealSlot, batchSize),
	}

	round := globals.GlobalRoundMap.GetRound(roundId)
	if round == nil {
		jww.INFO.Printf("skipping round %s, because it's dead", roundId)
		return
	}

	// Iterate over the input slots
	for i := uint64(0); i < batchSize; i++ {
		out := input.Slots[i]
		// Convert to PrecompRevealSlot
		msgSlot := &pb.PrecompRevealSlot{
			Slot: out.Slot,
			PartialMessageCypherText:        out.PartialMessageCypherText,
			PartialAssociatedDataCypherText: round.LastNode.AssociatedDataCypherText[i].Bytes(),
		}

		// Save the Message Precomputation
		round.LastNode.EncryptedMessagePrecomputation[i].SetBytes(
			out.EncryptedMessageKeys)

		// Append the PrecompRevealSlot to the PrecompRevealMessage
		msg.Slots[i] = msgSlot
	}

	// Send the first PrecompRevealMessage
	// Send the first PrecompPermute Message
	sendTime := time.Now()
	jww.INFO.Printf("[Last Node] Sending PrecompReveal Messages to %v at %s",
		NextServer, sendTime.Format(time.RFC3339))
	node.SendPrecompReveal(NextServer, msg)

	endTime := time.Now()
	jww.INFO.Printf("[Last Node] Finished Initializing "+
		"PrecompReveal(RoundId: %s, Phase: %s) in %d ms",
		input.RoundID, globals.Phase(input.LastOp).String(),
		(endTime.Sub(startTime))/time.Millisecond)
}

// TransmissionHandler for PrecompEncryptMessages
func (h PrecompEncryptHandler) Handler(
	roundId string, batchSize uint64, slots []*services.Slot) {
	startTime := time.Now()

	elapsed := startTime.Sub(globals.GlobalRoundMap.GetRound(roundId).
		CryptopStartTimes[globals.PRECOMP_ENCRYPT])

	jww.DEBUG.Printf("PrecompEncrypt Crypto took %v ms for "+
		"RoundId %s", elapsed, roundId)

	jww.INFO.Printf("Starting PrecompEncrypt.Handler(RoundId: %s) at %s",
		roundId, startTime.Format(time.RFC3339))

	// Create the PrecompEncryptMessage
	msg := &pb.PrecompEncryptMessage{
		RoundID: roundId,
		LastOp:  int32(globals.PRECOMP_ENCRYPT),
		Slots:   make([]*pb.PrecompEncryptSlot, batchSize),
	}

	// Iterate over the output channel
	for i := uint64(0); i < batchSize; i++ {
		// Type assert Slot to SlotEncrypt
		out := (*slots[i]).(*precomputation.PrecomputationSlot)
		// Convert to PrecompEncryptSlot
		msgSlot := &pb.PrecompEncryptSlot{
			Slot:                     out.Slot,
			EncryptedMessageKeys:     out.MessageCypher.Bytes(),
			PartialMessageCypherText: out.MessagePrecomputation.Bytes(),
		}

		// Put it into the slice
		msg.Slots[i] = msgSlot
	}

	// Advance internal state to PRECOMP_DECRYPT (the next phase)
	globals.GlobalRoundMap.SetPhase(roundId, globals.PRECOMP_REVEAL)

	sendTime := time.Now()
	if id.IsLastNode {
		// Transition to PrecompReveal phase
		jww.INFO.Printf("Starting PrecompReveal Phase to %v at %s",
			NextServer, sendTime.Format(time.RFC3339))
		precompEncryptLastNode(roundId, batchSize, msg)
	} else {
		// Send the completed PrecompEncryptMessage
		jww.INFO.Printf("Sending PrecompDecrypt Message to %v at %s",
			NextServer, sendTime.Format(time.RFC3339))
		node.SendPrecompEncrypt(NextServer, msg)
	}

	endTime := time.Now()
	jww.INFO.Printf("Finished PrecompEncrypt.Handler(RoundId: %s) in %d ms",
		roundId, (endTime.Sub(startTime))/time.Millisecond)
}
