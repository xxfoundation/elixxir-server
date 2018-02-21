////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/cryptops/realtime"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
	"testing"
)

func TestRealtimePermute(t *testing.T) {
	// Create a new Round
	roundId := "test"
	round := globals.NewRound(1)
	// Add round to the GlobalRoundMap
	globals.GlobalRoundMap.AddRound(roundId, round)

	// Create the test channels
	chIn := make(chan *services.Slot, round.BatchSize)
	chOut := make(chan *services.Slot, round.BatchSize)

	// Add the InChannel from the controller to round
	round.AddChannel(globals.REAL_PERMUTE, chIn)
	// Kick off RealtimePermute Transmission Handler
	services.BatchTransmissionDispatch(roundId, round.BatchSize,
		chOut, RealtimePermuteHandler{})

	// Create a slot to pass into the TransmissionHandler
	var slot services.Slot = &realtime.SlotPermute{
		Slot:                 uint64(0),
		EncryptedMessage:     cyclic.NewInt(12),
		EncryptedRecipientID: cyclic.NewInt(7),
	}

	// Pass slot as input to Permute's TransmissionHandler
	chOut <- &slot

	// Which should be populated into chIn once received
	received := <-chIn

	// Convert type for comparison
	expected := slot.(*realtime.SlotPermute)
	actual := (*received).(*realtime.SlotPermute)

	// Compare actual/expected
	if expected.Slot != actual.Slot {
		t.Errorf("Slot does not match!")
	}
	if expected.EncryptedMessage.Cmp(
		actual.EncryptedMessage) != 0 {
		t.Errorf("EncryptedMessage does not match!"+
			" Got %v, expected %v.",
			actual.EncryptedMessage.Text(10),
			expected.EncryptedMessage.Text(10))
	}
	if expected.EncryptedRecipientID.Cmp(
		actual.EncryptedRecipientID) != 0 {
		t.Errorf("EncryptedRecipientID does not match!"+
			" Got %v, expected %v.",
			actual.EncryptedRecipientID.Text(10),
			expected.EncryptedRecipientID.Text(10))
	}
}
