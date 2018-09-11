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
	"gitlab.com/privategrity/crypto/id"
)

func TestRealtimePeel(t *testing.T) {
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
	round.AddChannel(globals.REAL_PEEL, chIn)
	// Kick off RealtimeEncrypt Transmission Handler
	services.BatchTransmissionDispatch(roundId, round.BatchSize,
		chOut, RealtimeEncryptHandler{})

	// Create a slot to pass into the TransmissionHandler
	var slot services.Slot = &realtime.Slot{
		Slot:               uint64(0),
		CurrentID:          id.NewUserIDFromUint(42, t),
		Message:            cyclic.NewInt(7),
		EncryptedRecipient: cyclic.NewInt(42),
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
	if expected.CurrentID != actual.CurrentID {
		t.Errorf("CurrentID does not match!"+
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
}

// Smoke test
func TestRealtimePeelHandler_Handler(t *testing.T) {
	// Create a new Round
	roundId := "test"
	round := globals.NewRound(1)
	globals.InitLastNode(round)
	globals.IsLastNode = true
	// Add round to the GlobalRoundMap
	globals.GlobalRoundMap.AddRound(roundId, round)

	handler := RealtimePeelHandler{}
	userId := id.NewUserIDFromUint(1, t)
	s := make([]*services.Slot, 1)
	sl := &realtime.Slot{
		EncryptedRecipient: cyclic.NewInt(10),
		Message:            cyclic.NewInt(5),
		CurrentID:          userId,
		CurrentKey:         cyclic.NewInt(20),
	}
	slot := services.Slot(sl)
	s[0] = &slot

	// MIC verify the slot
	round.MIC_Verification[sl.Slot] = true
	// User registry must be initialized
	globals.Users = globals.NewUserRegistry("", "", "", "")
	globals.PopulateDummyUsers()

	handler.Handler(roundId, 1, s)

	phase := round.GetPhase()
	if phase != globals.REAL_COMPLETE {
		t.Errorf("RealtimePeelHandler: Realtime did not finish!")
	}
}
