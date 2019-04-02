////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/cryptops/realtime"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/services"
	"testing"
)

func TestRealtimeEncrypt(t *testing.T) {
	// Create a new Round
	roundId := "test"
	round := globals.NewRound(1, globals.GetGroup())
	globals.InitLastNode(round, globals.GetGroup())
	id.IsLastNode = true
	// Add round to the GlobalRoundMap
	globals.GlobalRoundMap.AddRound(roundId, round)

	// Create the test channels
	chIn := make(chan *services.Slot, round.BatchSize)
	chOut := make(chan *services.Slot, round.BatchSize)

	// Add the InChannel from the controller to round
	round.AddChannel(globals.REAL_ENCRYPT, chIn)
	// Kick off RealtimeEncrypt Transmission Handler
	services.BatchTransmissionDispatch(roundId, round.BatchSize,
		chOut, RealtimeIdentifyHandler{})
	round.LastNode.EncryptedMessage[0] = globals.GetGroup().NewInt(7)
	// Create a slot to pass into the TransmissionHandler
	userId := id.NewUserFromUint(42, t)
	associatedData := format.NewAssociatedData()
	associatedData.SetRecipient(userId)
	var slot services.Slot = &realtime.Slot{
		Slot:           uint64(0),
		CurrentID:      userId,
		Message:        globals.GetGroup().NewInt(7),
		AssociatedData: globals.GetGroup().NewIntFromBytes(associatedData.SerializeAssociatedData()),
	}

	// Pass slot as input to Encrypt's TransmissionHandler
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
	if *expected.CurrentID != *actual.CurrentID {
		t.Errorf("CurrentID does not match!"+
			" Got %q, expected %q.",
			*actual.CurrentID,
			*expected.CurrentID)
	}
	if expected.Message.Text(10) !=
		actual.Message.Text(10) {
		t.Errorf("EncryptedMessage does not match!"+
			" Got %v, expected %v.",
			actual.Message.Text(10),
			expected.Message.Text(10))
	}
}
