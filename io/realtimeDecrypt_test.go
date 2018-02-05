package io

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/cryptops/realtime"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
	"testing"
)

func TestRealtimeDecrypt(t *testing.T) {
	// Create a new Round
	roundId := "test"
	round := globals.NewRound(1)
	// Add round to the GlobalRoundMap
	globals.GlobalRoundMap.AddRound(roundId, round)

	// Create the test channels
	chIn := make(chan *services.Slot, round.BatchSize)
	chOut := make(chan *services.Slot, round.BatchSize)

	// Add the InChannel from the controller to round
	round.AddChannel(globals.REAL_DECRYPT, chIn)
	// Kick off RealtimeDecrypt Transmission Handler
	services.BatchTransmissionDispatch(roundId, round.BatchSize,
		chOut, RealtimeDecryptHandler{})

	// Create a slot to pass into the TransmissionHandler
	var slot services.Slot = &realtime.SlotDecryptOut{
		Slot:                 uint64(0),
		SenderID:             uint64(42),
		EncryptedMessage:     cyclic.NewInt(7),
		EncryptedRecipientID: cyclic.NewInt(3),
	}

	// Pass slot as input to Decrypt's TransmissionHandler
	chOut <- &slot

	// Which should be populated into chIn once received
	received := <-chIn

	// Convert type for comparison
	expected := slot.(*realtime.SlotDecryptOut)
	actual := (*received).(*realtime.SlotDecryptIn)

	// Compare actual/expected
	if expected.Slot != actual.Slot {
		t.Errorf("Slot does not match!")
	}
	if expected.SenderID != actual.SenderID {
		t.Errorf("SenderID does not match!"+
			" Got %v, expected %v.",
			actual.SenderID,
			expected.SenderID)
	}
	if expected.EncryptedMessage.Text(10) !=
		actual.EncryptedMessage.Text(10) {
		t.Errorf("EncryptedMessage does not match!"+
			" Got %v, expected %v.",
			actual.EncryptedMessage.Text(10),
			expected.EncryptedMessage.Text(10))
	}
	if expected.EncryptedRecipientID.Text(10) !=
		actual.EncryptedRecipientID.Text(10) {
		t.Errorf("EncryptedRecipientID does not match!"+
			" Got %v, expected %v.",
			actual.EncryptedRecipientID.Text(10),
			expected.EncryptedRecipientID.Text(10))
	}
}
