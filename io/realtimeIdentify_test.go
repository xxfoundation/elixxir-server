////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/elixxir/server/cryptops/realtime"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/services"
	"testing"

	"gitlab.com/elixxir/primitives/id"
)

func TestRealtimeIdentify(t *testing.T) {
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
	round.AddChannel(globals.REAL_IDENTIFY, chIn)
	// Kick off RealtimeIdentify Transmission Handler
	services.BatchTransmissionDispatch(roundId, round.BatchSize,
		chOut, RealtimePermuteHandler{})

	// Create a slot to pass into the TransmissionHandler
	var slot services.Slot = &realtime.Slot{
		Slot:           uint64(0),
		Message:        globals.GetGroup().NewInt(12),
		AssociatedData: globals.GetGroup().NewInt(7),
		CurrentKey:     globals.GetGroup().NewInt(1),
	}

	// Pass slot as input to Identify's TransmissionHandler
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
	if expected.AssociatedData.Cmp(
		actual.AssociatedData) != 0 {
		t.Errorf("AssociatedData does not match!"+
			" Got %v, expected %v.",
			actual.AssociatedData.Text(10),
			expected.AssociatedData.Text(10))
	}
}
