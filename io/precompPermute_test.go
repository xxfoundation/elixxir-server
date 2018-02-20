package io

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/cryptops/precomputation"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
	"testing"
)

func TestPrecompPermute(t *testing.T) {
	// Create a new Round
	roundId := "test"
	round := globals.NewRound(1)
	// Add round to the GlobalRoundMap
	globals.GlobalRoundMap.AddRound(roundId, round)

	// Create the test channels
	chIn := make(chan *services.Slot, round.BatchSize)
	chOut := make(chan *services.Slot, round.BatchSize)

	// Add the InChannel from the controller to round
	round.AddChannel(globals.PRECOMP_PERMUTE, chIn)
	// Kick off PrecompPermute Transmission Handler
	services.BatchTransmissionDispatch(roundId, round.BatchSize,
		chOut, PrecompPermuteHandler{})

	// Create a slot to pass into the TransmissionHandler
	var slot services.Slot = &precomputation.PrecomputationSlot{
		Slot:                      uint64(0),
		MessageCypher:             cyclic.NewInt(12),
		RecipientIDCypher:         cyclic.NewInt(7),
		MessagePrecomputation:     cyclic.NewInt(3),
		RecipientIDPrecomputation: cyclic.NewInt(8)}

	// Pass slot as input to Permute's TransmissionHandler
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
	if expected.MessageCypher.Cmp(
		actual.MessageCypher) != 0 {
		t.Errorf("MessageCypher does not match!"+
			" Got %v, expected %v.",
			actual.MessageCypher.Text(10),
			expected.MessageCypher.Text(10))
	}
	if expected.RecipientIDCypher.Cmp(
		actual.RecipientIDCypher) != 0 {
		t.Errorf("RecipientIDCypher does not match!"+
			" Got %v, expected %v.",
			actual.RecipientIDCypher.Text(10),
			expected.RecipientIDCypher.Text(10))
	}
	if expected.MessagePrecomputation.Cmp(
		actual.MessagePrecomputation) != 0 {
		t.Errorf("MessagePrecomputation does not match!"+
			" Got %v, expected %v.",
			actual.MessagePrecomputation.Text(10),
			expected.MessagePrecomputation.Text(10))
	}
	if expected.RecipientIDPrecomputation.Cmp(
		actual.RecipientIDPrecomputation) != 0 {
		t.Errorf("RecipientIDPrecomputation does not match!"+
			" Got %v, expected %v.",
			actual.RecipientIDPrecomputation.Text(10),
			expected.RecipientIDPrecomputation.Text(10))
	}
}
