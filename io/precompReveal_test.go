////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package io

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/cryptops/precomputation"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
	"testing"
)

func TestPrecompReveal(t *testing.T) {
	// Create a new Round
	roundId := "test"
	round := globals.NewRound(1)
	// Add round to the GlobalRoundMap
	globals.GlobalRoundMap.AddRound(roundId, round)

	// Create the test channels
	chIn := make(chan *services.Slot, round.BatchSize)
	chOut := make(chan *services.Slot, round.BatchSize)

	// Add the InChannel from the controller to round
	round.AddChannel(globals.PRECOMP_REVEAL, chIn)
	// Kick off PrecompReveal Transmission Handler
	services.BatchTransmissionDispatch(roundId, round.BatchSize,
		chOut, PrecompRevealHandler{})

	// Create a slot to pass into the TransmissionHandler
	var slot services.Slot = &precomputation.PrecomputationSlot{
		Slot: uint64(0),
		MessagePrecomputation:     cyclic.NewInt(3),
		RecipientIDPrecomputation: cyclic.NewInt(10),
	}

	// Pass slot as input to Reveal's TransmissionHandler
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
	if expected.MessagePrecomputation.Text(10) !=
		actual.MessagePrecomputation.Text(10) {
		t.Errorf("MessagePrecomputation does not match!"+
			" Got %v, expected %v.",
			actual.MessagePrecomputation.Text(10),
			expected.MessagePrecomputation.Text(10))
	}
}
