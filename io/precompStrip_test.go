////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/elixxir/server/cryptops/precomputation"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/services"
	"testing"
)

func TestPrecompStrip(t *testing.T) {
	// Create a new Round
	roundId := "test"
	round := globals.NewRound(1, globals.GetGroup())
	globals.InitLastNode(round, globals.GetGroup())
	globals.IsLastNode = true
	// Add round to the GlobalRoundMap
	globals.GlobalRoundMap.AddRound(roundId, round)

	// Create the test channels
	chIn := make(chan *services.Slot, round.BatchSize)
	chOut := make(chan *services.Slot, round.BatchSize)

	// Add the InChannel from the controller to round
	round.AddChannel(globals.PRECOMP_STRIP, chIn)
	// Kick off PrecompStrip Transmission Handler
	services.BatchTransmissionDispatch(roundId, round.BatchSize,
		chOut, PrecompRevealHandler{})

	// Create a slot to pass into the TransmissionHandler
	var slot services.Slot = &precomputation.PrecomputationSlot{
		Slot:                         uint64(0),
		MessageCypher:                globals.GetGroup().NewInt(12),
		AssociatedDataCypher:         globals.GetGroup().NewInt(7),
		MessagePrecomputation:        globals.GetGroup().NewInt(3),
		AssociatedDataPrecomputation: globals.GetGroup().NewInt(8)}

	// Pass slot as input to Strip's TransmissionHandler
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
	if expected.MessagePrecomputation.Text(10) !=
		actual.MessagePrecomputation.Text(10) {
		t.Errorf("MessagePrecomputation does not match!"+
			" Got %v, expected %v.",
			actual.MessagePrecomputation.Text(10),
			expected.MessagePrecomputation.Text(10))
	}
	if expected.AssociatedDataPrecomputation.Text(10) !=
		actual.AssociatedDataPrecomputation.Text(10) {
		t.Errorf("AssociatedDataPrecomputation does not match!"+
			" Got %v, expected %v.",
			actual.AssociatedDataPrecomputation.Text(10),
			expected.AssociatedDataPrecomputation.Text(10))
	}
}

// Smoke test
func TestPrecompStripHandler_Handler(t *testing.T) {
	// Create a new Round
	roundId := "test"
	round := globals.NewRound(1, globals.GetGroup())
	globals.InitLastNode(round, globals.GetGroup())
	globals.IsLastNode = true
	// Add round to the GlobalRoundMap
	globals.GlobalRoundMap.AddRound(roundId, round)

	handler := PrecompStripHandler{}
	s := make([]*services.Slot, 1)
	sl := &precomputation.PrecomputationSlot{
		MessagePrecomputation:        globals.GetGroup().NewInt(10),
		AssociatedDataPrecomputation: globals.GetGroup().NewInt(10),
	}
	slot := services.Slot(sl)
	s[0] = &slot
	handler.Handler(roundId, 1, s)

	phase := round.GetPhase()
	if phase != globals.PRECOMP_COMPLETE {
		t.Errorf("PrecompStripHandler: Precomp did not finish!")
	}
	if round.LastNode.MessagePrecomputation[0].Cmp(
		sl.MessagePrecomputation) != 0 {
		t.Errorf("PrecompStripHandler: Failed to save" +
			" MessagePrecomputation!")
	}
	if round.LastNode.AssociatedDataPrecomputation[0].Cmp(
		sl.AssociatedDataPrecomputation) != 0 {
		t.Errorf("PrecompStripHandler: Failed to save" +
			" AssociatedDataPrecomputation!")
	}
}
