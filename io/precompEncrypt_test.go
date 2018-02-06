package io

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/cryptops/precomputation"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
	"testing"
)

func TestPrecompEncrypt(t *testing.T) {
	// Create a new Round
	roundId := "test"
	round := globals.NewRound(1)
	// Add round to the GlobalRoundMap
	globals.GlobalRoundMap.AddRound(roundId, round)

	// Create the test channels
	chIn := make(chan *services.Slot, round.BatchSize)
	chOut := make(chan *services.Slot, round.BatchSize)

	// Add the InChannel from the controller to round
	round.AddChannel(globals.PRECOMP_ENCRYPT, chIn)
	// Kick off PrecompEncrypt Transmission Handler
	services.BatchTransmissionDispatch(roundId, round.BatchSize,
		chOut, PrecompEncryptHandler{})

	// Create a slot to pass into the TransmissionHandler
	var slot services.Slot = &precomputation.SlotEncrypt{
		Slot:                     uint64(0),
		EncryptedMessageKeys:     cyclic.NewInt(12),
		PartialMessageCypherText: cyclic.NewInt(3),
	}

	// Pass slot as input to Encrypt's TransmissionHandler
	chOut <- &slot

	// Which should be populated into chIn once received
	received := <-chIn

	// Convert type for comparison
	expected := slot.(*precomputation.SlotEncrypt)
	actual := (*received).(*precomputation.SlotEncrypt)

	// Compare actual/expected
	if expected.Slot != actual.Slot {
		t.Errorf("Slot does not match!")
	}
	if expected.EncryptedMessageKeys.Text(10) !=
		actual.EncryptedMessageKeys.Text(10) {
		t.Errorf("EncryptedMessageKeys does not match!"+
			" Got %v, expected %v.",
			actual.EncryptedMessageKeys.Text(10),
			expected.EncryptedMessageKeys.Text(10))
	}
	if expected.PartialMessageCypherText.Text(10) !=
		actual.PartialMessageCypherText.Text(10) {
		t.Errorf("PartialMessageCypherText does not match!"+
			" Got %v, expected %v.",
			actual.PartialMessageCypherText.Text(10),
			expected.PartialMessageCypherText.Text(10))
	}
}
