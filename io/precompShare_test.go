package io

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/cryptops/precomputation"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
	"testing"
)

func TestPrecompShare(t *testing.T) {
	// Create a new Round
	roundId := "test"
	round := globals.NewRound(1)
	// Add round to the GlobalRoundMap
	globals.GlobalRoundMap.AddRound(roundId, round)

	// Create the test channels
	chIn := make(chan *services.Slot, round.BatchSize)
	chOut := make(chan *services.Slot, round.BatchSize)

	// Add the InChannel from the controller to round
	round.AddChannel(globals.PRECOMP_SHARE, chIn)
	// Kick off PrecompShare Transmission Handler
	services.BatchTransmissionDispatch(roundId, round.BatchSize,
		chOut, PrecompShareHandler{})

	// Create a slot to pass into the TransmissionHandler
	var slot services.Slot = &precomputation.SlotShare{
		Slot: uint64(0),
		PartialRoundPublicCypherKey: cyclic.NewInt(3),
	}

	// Pass slot as input to Share's TransmissionHandler
	chOut <- &slot

	// Which should be populated into chIn once received
	received := <-chIn

	// Convert type for comparison
	expected := slot.(*precomputation.SlotShare)
	actual := (*received).(*precomputation.SlotShare)

	// Compare actual/expected
	if expected.Slot != actual.Slot {
		t.Errorf("Slot does not match!")
	}
	if expected.PartialRoundPublicCypherKey.Text(10) !=
		actual.PartialRoundPublicCypherKey.Text(10) {
		t.Errorf("PartialRoundPublicCypherKey does not match!"+
			" Got %v, expected %v.",
			actual.PartialRoundPublicCypherKey.Text(10),
			expected.PartialRoundPublicCypherKey.Text(10))
	}
}
