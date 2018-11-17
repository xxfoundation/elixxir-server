////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/id"
	"gitlab.com/elixxir/server/cryptops/realtime"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/services"
	"testing"
)

func TestRealtimePermute(t *testing.T) {
	// Create a new Round
	roundId := "test"
	round := globals.NewRound(1)
	globals.InitLastNode(round)
	globals.IsLastNode = true
	// Add round to the GlobalRoundMap
	globals.GlobalRoundMap.AddRound(roundId, round)

	// Create the test channels
	chIn := make(chan *services.Slot, round.BatchSize)
	chOut := make(chan *services.Slot, round.BatchSize)

	// Add the InChannel from the controller to round
	round.AddChannel(globals.REAL_PERMUTE, chIn)
	// Kick off RealtimePermute Transmission Handler
	services.BatchTransmissionDispatch(roundId, round.BatchSize,
		chOut, RealtimeDecryptHandler{})

	// Create a slot to pass into the TransmissionHandler
	var slot services.Slot = &realtime.Slot{
		Slot:               uint64(0),
		EncryptedRecipient: cyclic.NewInt(7),
		Message:            cyclic.NewInt(12),
		// TODO Should this really need to be populated? Will it always be
		// populated in real usage?
		CurrentID:  id.NewUserIDFromUint(5, t),
		CurrentKey: cyclic.NewInt(1),
	}

	// Pass slot as input to Permute's TransmissionHandler
	chOut <- &slot

	// Which should be populated into chIn once received
	received := <-chIn

	// Convert type for comparison
	expected := slot.(*realtime.Slot)
	actual := (*received).(*realtime.Slot)

	// Compare actual/expected
	if expected.Slot != actual.Slot {
		t.Errorf("Slot does not match!")
	}
	if expected.Message.Cmp(
		actual.Message) != 0 {
		t.Errorf("EncryptedMessage does not match!"+
			" Got %v, expected %v.",
			actual.Message.Text(10),
			expected.Message.Text(10))
	}
	if expected.EncryptedRecipient.Cmp(
		actual.EncryptedRecipient) != 0 {
		t.Errorf("EncryptedRecipientID does not match!"+
			" Got %v, expected %v.",
			actual.EncryptedRecipient.Text(10),
			expected.EncryptedRecipient.Text(10))
	}
}
