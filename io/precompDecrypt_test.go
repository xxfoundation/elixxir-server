package io

import (
	"gitlab.com/privategrity/comms/mixserver"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/cryptops/precomputation"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
	"testing"
)

func TestPrecompDecrypt(t *testing.T) {
	// Initialize GlobalRoundMap
	globals.GlobalRoundMap = globals.NewRoundMap()
	// Kick off comms server
	localServer := "localhost:5556"
	go mixserver.StartServer(localServer,
		ServerImpl{Rounds: &globals.GlobalRoundMap})
	// Next hop will be back to us
	NextServer = localServer

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
		chOut, PrecompDecryptHandler{})

	// Create a slot to pass into the TransmissionHandler
	var slot services.Slot = &precomputation.SlotDecrypt{
		Slot:                         uint64(0),
		EncryptedMessageKeys:         cyclic.NewInt(12),
		EncryptedRecipientIDKeys:     cyclic.NewInt(7),
		PartialMessageCypherText:     cyclic.NewInt(3),
		PartialRecipientIDCypherText: cyclic.NewInt(8)}

	// Pass slot as input to Decrypt's TransmissionHandler
	chOut <- &slot

	// Which should be populated into chIn once received
	received := <-chIn

	// Convert type for comparison
	expected := slot.(*precomputation.SlotDecrypt)
	actual := (*received).(*precomputation.SlotDecrypt)

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
	if expected.EncryptedRecipientIDKeys.Text(10) !=
		actual.EncryptedRecipientIDKeys.Text(10) {
		t.Errorf("EncryptedRecipientIDKeys does not match!"+
			" Got %v, expected %v.",
			actual.EncryptedRecipientIDKeys.Text(10),
			expected.EncryptedRecipientIDKeys.Text(10))
	}
	if expected.PartialMessageCypherText.Text(10) !=
		actual.PartialMessageCypherText.Text(10) {
		t.Errorf("PartialMessageCypherText does not match!"+
			" Got %v, expected %v.",
			actual.PartialMessageCypherText.Text(10),
			expected.PartialMessageCypherText.Text(10))
	}
	if expected.PartialRecipientIDCypherText.Text(10) !=
		actual.PartialRecipientIDCypherText.Text(10) {
		t.Errorf("PartialRecipientIDCypherText does not match!"+
			" Got %v, expected %v.",
			actual.PartialRecipientIDCypherText.Text(10),
			expected.PartialRecipientIDCypherText.Text(10))
	}
}
