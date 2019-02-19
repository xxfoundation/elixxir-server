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
type PrecompShareHandler struct{}

// ReceptionHandler for PrecompShareMessages
func (s ServerImpl) PrecompShare(input *pb.PrecompShareMessage) {
	startTime := time.Now()
	jww.INFO.Printf("Starting PrecompShare(RoundId: %s, Phase: %s) at %s",
		input.RoundID, globals.Phase(input.LastOp).String(),
		startTime.Format(time.RFC3339))

	// Store when the operation started
	globals.GlobalRoundMap.GetRound(input.RoundID).CryptopStartTimes[globals.
		PRECOMP_SHARE] = startTime

	// Get the input channel for the cryptop
	chIn := s.GetChannel(input.RoundID, globals.PRECOMP_SHARE)
	// Iterate through the Slots in the PrecompShareMessage
	for i := 0; i < len(input.Slots); i++ {
		// Convert input message to equivalent SlotShare
		in := input.Slots[i]
		var slot services.Slot = &precomputation.SlotShare{
			Slot: in.Slot,
			PartialRoundPublicCypherKey: cyclic.NewIntFromBytes(
				in.PartialRoundPublicCypherKey),
		}
		// Pass slot as input to Share's channel
		chIn <- &slot
	}

	close(chIn)

	endTime := time.Now()
	jww.INFO.Printf("Finished PrecompShare(RoundId: %s, Phase: %s) in %d ms",
		input.RoundID, globals.Phase(input.LastOp).String(),
		(endTime.Sub(startTime))/time.Millisecond)
}

// TODO finish implementing this stubbed-out method
func (s ServerImpl) PrecompShareCompare(*pb.PrecompShareCompareMessage) {}

// TODO finish implementing this stubbed-out method
func (s ServerImpl) PrecompShareConfirm(*pb.PrecompShareConfirmMessage) {}

// TODO finish implementing this stubbed-out method
func (s ServerImpl) PrecompShareInit(*pb.PrecompShareInitMessage) {}

// Transition to PrecompDecrypt phase on the last node
func precompShareLastNode(roundId string, input *pb.PrecompShareMessage) {

	startTime := time.Now()
	jww.INFO.Printf("[Last Node] Initializing PrecompDecrypt(RoundId: %s, "+
		"Phase: %s) at %s",
		input.RoundID, globals.Phase(input.LastOp).String(),
		startTime.Format(time.RFC3339))

	// Store when the operation started
	globals.GlobalRoundMap.GetRound(input.RoundID).CryptopStartTimes[globals.
		PRECOMP_SHARE] = startTime

	// TODO: record the start precomp decrypt time for this round here,
	//       and print the time it took for the Decrypt phase to complete.

	// Force batchSize to be the same as the round
	// as the batchSize we need may be inconsistent
	// with the Share phase batchSize
	round := globals.GlobalRoundMap.GetRound(roundId)
	if round == nil {
		jww.INFO.Printf("skipping round %s, because it's dead", roundId)
		return
	}
	batchSize := round.BatchSize

	// For each node, set CypherPublicKey to
	// shareResult.PartialRoundPublicCypherKey
	jww.INFO.Println("Setting node Public Keys...")
	for i := range Servers {
		node.SetPublicKey(Servers[i], &pb.PublicKeyMessage{
			RoundID:   input.RoundID,
			PublicKey: input.Slots[0].PartialRoundPublicCypherKey,
		})
	}

	// Create the PrecompDecryptMessage
	msg := &pb.PrecompDecryptMessage{
		RoundID: roundId,
		LastOp:  int32(globals.PRECOMP_SHARE),
		Slots:   make([]*pb.PrecompDecryptSlot, batchSize),
	}

	// Iterate over the input slots
	for i := uint64(0); i < batchSize; i++ {
		// Convert to PrecompDecryptSlot
		msgSlot := &pb.PrecompDecryptSlot{
			Slot:                         uint64(i),
			EncryptedMessageKeys:         cyclic.NewInt(1).Bytes(),
			PartialMessageCypherText:     cyclic.NewInt(1).Bytes(),
			EncryptedRecipientIDKeys:     cyclic.NewInt(1).Bytes(),
			PartialRecipientIDCypherText: cyclic.NewInt(1).Bytes(),
		}
		msg.Slots[i] = msgSlot
	}

	// Send first PrecompDecrypt Message
	sendTime := time.Now()
	jww.INFO.Printf("[Last Node] Sending PrecompDecrypt Messages to %v at %s",
		NextServer, sendTime.Format(time.RFC3339))
	node.SendPrecompDecrypt(NextServer, msg)

	endTime := time.Now()
	jww.INFO.Printf("[Last Node] Finished Initializing "+
		"PrecompDecrypt(RoundId: %s, Phase: %s) in %d ms",
		input.RoundID, globals.Phase(input.LastOp).String(),
		(endTime.Sub(startTime))/time.Millisecond)
}

// TransmissionHandler for PrecompShareMessages
func (h PrecompShareHandler) Handler(
	roundId string, batchSize uint64, slots []*services.Slot) {
	startTime := time.Now()
	jww.INFO.Printf("Starting PrecompShare.Handler(RoundId: %s) at %s",
		roundId, startTime.Format(time.RFC3339))

	elapsed := startTime.Sub(globals.GlobalRoundMap.GetRound(roundId).
		CryptopStartTimes[globals.PRECOMP_SHARE])

	jww.DEBUG.Printf("PrecompShare Crypto took %v ms for "+
		"RoundId %s", elapsed, roundId)

	// Create the PrecompShareMessage
	msg := &pb.PrecompShareMessage{
		RoundID: roundId,
		LastOp:  int32(globals.PRECOMP_SHARE),
		Slots:   make([]*pb.PrecompShareSlot, batchSize),
	}

	// Iterate over the output channel
	for i := uint64(0); i < batchSize; i++ {
		// Type assert Slot to SlotShare
		out := (*slots[i]).(*precomputation.SlotShare)
		// Convert to PrecompShareSlot
		msgSlot := &pb.PrecompShareSlot{
			Slot: out.Slot,
			PartialRoundPublicCypherKey: out.
				PartialRoundPublicCypherKey.Bytes(),
		}

		// Put it into the slice
		msg.Slots[i] = msgSlot
	}

	sendTime := time.Now()
	// Returns whether this is the first time Share is being run TODO Something better
	IsFirstRun := (*slots[0]).(*precomputation.SlotShare).PartialRoundPublicCypherKey.Cmp(globals.Grp.G) == 0

	// Advance internal state to the next phase
	if id.IsLastNode && IsFirstRun {
		globals.GlobalRoundMap.SetPhase(roundId, globals.PRECOMP_SHARE)
	} else {
		globals.GlobalRoundMap.SetPhase(roundId, globals.PRECOMP_DECRYPT)
	}

	if id.IsLastNode && !IsFirstRun {
		// Transition to PrecompDecrypt phase
		// if we are last node and this isn't the first run
		jww.INFO.Printf("Starting PrecompDecrypt Phase to %v at %s",
			NextServer, sendTime.Format(time.RFC3339))
		precompShareLastNode(roundId, msg)
	} else {
		// Send the completed PrecompShareMessage
		jww.INFO.Printf("Sending PrecompShare Message to %v at %s",
			NextServer, sendTime.Format(time.RFC3339))
		node.SendPrecompShare(NextServer, msg)
	}

	endTime := time.Now()
	jww.INFO.Printf("Finished PrecompShare.Handler(RoundId: %s) in %d ms",
		roundId, (endTime.Sub(startTime))/time.Millisecond)
}
