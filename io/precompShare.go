////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/privategrity/comms/clusterclient"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/cryptops/precomputation"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
)

// Blank struct for implementing services.BatchTransmission
type PrecompShareHandler struct{}

// ReceptionHandler for PrecompShareMessages
func (s ServerImpl) PrecompShare(input *pb.PrecompShareMessage) {
	jww.DEBUG.Printf("Received PrecompShare Message %v from phase %s...",
		input.RoundID, globals.Phase(input.LastOp).String())
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
}

// TODO finish implementing this stubbed-out method
func (s ServerImpl) PrecompShareCompare(*pb.PrecompShareCompareMessage) {}

// TODO finish implementing this stubbed-out method
func (s ServerImpl) PrecompShareConfirm(*pb.PrecompShareConfirmMessage) {}

// TODO finish implementing this stubbed-out method
func (s ServerImpl) PrecompShareInit(message *pb.PrecompShareInitMessage) {}

// Transition to PrecompDecrypt phase on the last node
func precompShareLastNode(roundId string, input *pb.PrecompShareMessage) {
	// Force batchSize to be the same as the round
	// as the batchSize we need may be inconsistent
	// with the Share phase batchSize
	batchSize := globals.GlobalRoundMap.GetRound(roundId).BatchSize

	// For each node, set CypherPublicKey to
	// shareResult.PartialRoundPublicCypherKey
	jww.INFO.Println("Setting node Public Keys...")
	for i := range Servers {
		clusterclient.SetPublicKey(Servers[i], &pb.PublicKeyMessage{
			RoundID:   input.RoundID,
			PublicKey: input.Slots[0].PartialRoundPublicCypherKey,
		})
	}

	// Kick off the PrecompDecrypt phase
	jww.INFO.Println("Beginning PrecompDecrypt Phase...")

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
	jww.DEBUG.Printf("Sending PrecompDecrypt Message to %v...", NextServer)
	clusterclient.SendPrecompDecrypt(NextServer, msg)
}

// TransmissionHandler for PrecompShareMessages
func (h PrecompShareHandler) Handler(
	roundId string, batchSize uint64, slots []*services.Slot) {
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

	// Returns whether this is the first time Share is being run TODO Something better
	IsFirstRun := (*slots[0]).(*precomputation.SlotShare).PartialRoundPublicCypherKey.Cmp(globals.Grp.G) == 0
	if IsLastNode && !IsFirstRun {
		// Transition to PrecompDecrypt phase
		// if we are last node and this isn't the first run
		precompShareLastNode(roundId, msg)
	} else {
		// Send the completed PrecompShareMessage
		jww.DEBUG.Printf("Sending PrecompShare Message to %v...", NextServer)
		clusterclient.SendPrecompShare(NextServer, msg)
	}
}
