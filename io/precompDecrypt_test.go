////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package io

import (
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/cryptops/precomputation"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/services"
	"testing"
)

type DummyPrecompDecryptHandler struct{}

func (h DummyPrecompDecryptHandler) Handler(
	roundId string, batchSize uint64, slots []*services.Slot) {
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
			Slot:                         out.Slot,
			EncryptedMessageKeys:         out.MessageCypher.Bytes(),
			EncryptedRecipientIDKeys:     out.RecipientIDCypher.Bytes(),
			PartialMessageCypherText:     out.MessagePrecomputation.Bytes(),
			PartialRecipientIDCypherText: out.RecipientIDPrecomputation.Bytes(),
		}

		// Append the PrecompDecryptSlot to the PrecompDecryptMessage
		msg.Slots[i] = msgSlot
	}

	// Advance internal state to PRECOMP_DECRYPT (the next phase)
	globals.GlobalRoundMap.SetPhase(roundId, globals.PRECOMP_DECRYPT)

	// Send the completed PrecompDecryptMessage
	node.SendPrecompDecrypt(NextServer, msg)
}

func TestPrecompDecrypt(t *testing.T) {
	// Create a new Round
	roundId := "test"
	round := globals.NewRound(1)
	// Add round to the GlobalRoundMap
	globals.GlobalRoundMap.AddRound(roundId, round)

	// Create the test channels
	chIn := make(chan *services.Slot, round.BatchSize)
	chOut := make(chan *services.Slot, round.BatchSize)

	// Add the InChannel from the controller to round
	round.AddChannel(globals.PRECOMP_DECRYPT, chIn)
	// Kick off PrecompDecrypt Transmission Handler
	services.BatchTransmissionDispatch(roundId, round.BatchSize,
		chOut, DummyPrecompDecryptHandler{})

	// Create a slot to pass into the TransmissionHandler
	var slot services.Slot = &precomputation.PrecomputationSlot{
		Slot:                      uint64(0),
		MessageCypher:             cyclic.NewInt(12),
		RecipientIDCypher:         cyclic.NewInt(7),
		MessagePrecomputation:     cyclic.NewInt(3),
		RecipientIDPrecomputation: cyclic.NewInt(8)}

	// Pass slot as input to Decrypt's TransmissionHandler
	chOut <- &slot

	// Which should be populated into chIn once received
	received := <-chIn

	// Convert type for comparison
	expected := slot.(*precomputation.PrecomputationSlot)
	actual := (*received).(*precomputation.PrecomputationSlot)

	// Compare actual/expected
	if expected.Slot != actual.Slot {
		t.Errorf("Slot does not match!")
	}
	if expected.MessageCypher.Text(10) !=
		actual.MessageCypher.Text(10) {
		t.Errorf("MessageCypher does not match!"+
			" Got %v, expected %v.",
			actual.MessageCypher.Text(10),
			expected.MessageCypher.Text(10))
	}
	if expected.RecipientIDCypher.Text(10) !=
		actual.RecipientIDCypher.Text(10) {
		t.Errorf("RecipientIDCypher does not match!"+
			" Got %v, expected %v.",
			actual.RecipientIDCypher.Text(10),
			expected.RecipientIDCypher.Text(10))
	}
	if expected.MessagePrecomputation.Text(10) !=
		actual.MessagePrecomputation.Text(10) {
		t.Errorf("MessagePrecomputation does not match!"+
			" Got %v, expected %v.",
			actual.MessagePrecomputation.Text(10),
			expected.MessagePrecomputation.Text(10))
	}
	if expected.RecipientIDPrecomputation.Text(10) !=
		actual.RecipientIDPrecomputation.Text(10) {
		t.Errorf("RecipientIDPrecomputation does not match!"+
			" Got %v, expected %v.",
			actual.RecipientIDPrecomputation.Text(10),
			expected.RecipientIDPrecomputation.Text(10))
	}
}
