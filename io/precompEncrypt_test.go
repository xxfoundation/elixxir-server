////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package io

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/cryptops/precomputation"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/services"
	"testing"
)

func TestPrecompEncrypt(t *testing.T) {
	// Create a new Round
	roundId := "test"
	round := globals.NewRound(1)
	globals.InitLastNode(round)
	// Add round to the GlobalRoundMap
	globals.GlobalRoundMap.AddRound(roundId, round)
	id.IsLastNode = true

	// Create the test channels
	chIn := make(chan *services.Slot, round.BatchSize)
	chOut := make(chan *services.Slot, round.BatchSize)

	// Add the InChannel from the controller to round
	round.AddChannel(globals.PRECOMP_ENCRYPT, chIn)
	// Kick off PrecompEncrypt Transmission Handler
	services.BatchTransmissionDispatch(roundId, round.BatchSize,
		chOut, PrecompPermuteHandler{})

	// Create a slot to pass into the TransmissionHandler
	var slot services.Slot = &precomputation.PrecomputationSlot{
		Slot:                         uint64(0),
		MessageCypher:                cyclic.NewInt(12),
		MessagePrecomputation:        cyclic.NewInt(3),
		AssociatedDataCypher:         cyclic.NewInt(1),
		AssociatedDataPrecomputation: cyclic.NewInt(1),
	}

	// Pass slot as input to Encrypt's TransmissionHandler
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
	if expected.MessagePrecomputation.Text(10) !=
		actual.MessagePrecomputation.Text(10) {
		t.Errorf("MessagePrecomputation does not match!"+
			" Got %v, expected %v.",
			actual.MessagePrecomputation.Text(10),
			expected.MessagePrecomputation.Text(10))
	}
}
