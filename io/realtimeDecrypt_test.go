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

func TestRealtimeDecrypt(t *testing.T) {
	// Create a new Round
	roundId := "test"
	round := globals.NewRound(1)
	globals.InitLastNode(round)
	globals.IsLastNode = true
	// Add round to the GlobalRoundMap
	globals.GlobalRoundMap.AddRound(roundId, round)

	// Create the test channels
	chIn := make(chan *services.Slot, round.BatchSize)

	// Add the InChannel from the controller to round
	round.AddChannel(globals.REAL_DECRYPT, chIn)

	// Create a slot to pass into the TransmissionHandler
	var slot services.Slot = &realtime.RealtimeSlot{
		Slot:               uint64(0),
		CurrentID:          uint64(42),
		Message:            cyclic.NewInt(7),
		EncryptedRecipient: cyclic.NewInt(3),
	}

	slots := [1]*realtime.RealtimeSlot{slot.(*realtime.RealtimeSlot)}
	NextServer = "localhost:5555"
	KickoffDecryptHandler(roundId, uint64(1), slots[:])

	// Which should be populated into chIn once received
	received := <-chIn

	// Convert type for comparison
	expected := slot.(*realtime.RealtimeSlot)
	actual := (*received).(*realtime.RealtimeSlot)

	// Compare actual/expected
	if expected.Slot != actual.Slot {
		t.Errorf("Slot does not match!")
	}
	if expected.CurrentID != actual.CurrentID {
		t.Errorf("SenderID does not match!"+
			" Got %v, expected %v.",
			actual.CurrentID,
			expected.CurrentID)
	}
	if expected.Message.Text(10) !=
		actual.Message.Text(10) {
		t.Errorf("EncryptedMessage does not match!"+
			" Got %v, expected %v.",
			actual.Message.Text(10),
			expected.Message.Text(10))
	}
	if expected.EncryptedRecipient.Text(10) !=
		actual.EncryptedRecipient.Text(10) {
		t.Errorf("EncryptedRecipientID does not match!"+
			" Got %v, expected %v.",
			actual.EncryptedRecipient.Text(10),
			expected.EncryptedRecipient.Text(10))
	}
}
