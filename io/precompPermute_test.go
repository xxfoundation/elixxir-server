////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/cryptops/precomputation"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/services"
	"testing"
)

func TestPrecompPermute(t *testing.T) {
	// Create a new Round
	roundId := "test"
	round := globals.NewRound(1)
	globals.InitLastNode(round)
	id.IsLastNode = true
	// Add round to the GlobalRoundMap
	globals.GlobalRoundMap.AddRound(roundId, round)

	// Create the test channels
	chIn := make(chan *services.Slot, round.BatchSize)
	chOut := make(chan *services.Slot, round.BatchSize)

	// Add the InChannel from the controller to round
	round.AddChannel(globals.PRECOMP_PERMUTE, chIn)
	// Kick off PrecompPermute Transmission Handler
	services.BatchTransmissionDispatch(roundId, round.BatchSize,
		chOut, PrecompDecryptHandler{})

	// Create a slot to pass into the TransmissionHandler
	var slot services.Slot = &precomputation.PrecomputationSlot{
		Slot:                         uint64(0),
		MessageCypher:                cyclic.NewInt(12),
		AssociatedDataCypher:         cyclic.NewInt(7),
		MessagePrecomputation:        cyclic.NewInt(3),
		AssociatedDataPrecomputation: cyclic.NewInt(8)}

	// Pass slot as input to Permute's TransmissionHandler
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
	if expected.MessageCypher.Cmp(
		actual.MessageCypher) != 0 {
		t.Errorf("MessageCypher does not match!"+
			" Got %v, expected %v.",
			actual.MessageCypher.Text(10),
			expected.MessageCypher.Text(10))
	}
	if expected.AssociatedDataCypher.Cmp(
		actual.AssociatedDataCypher) != 0 {
		t.Errorf("AssociatedDataCypher does not match!"+
			" Got %v, expected %v.",
			actual.AssociatedDataCypher.Text(10),
			expected.AssociatedDataCypher.Text(10))
	}
	if expected.MessagePrecomputation.Cmp(
		actual.MessagePrecomputation) != 0 {
		t.Errorf("MessagePrecomputation does not match!"+
			" Got %v, expected %v.",
			actual.MessagePrecomputation.Text(10),
			expected.MessagePrecomputation.Text(10))
	}
	if expected.AssociatedDataPrecomputation.Cmp(
		actual.AssociatedDataPrecomputation) != 0 {
		t.Errorf("AssociatedDataPrecomputation does not match!"+
			" Got %v, expected %v.",
			actual.AssociatedDataPrecomputation.Text(10),
			expected.AssociatedDataPrecomputation.Text(10))
	}
}
