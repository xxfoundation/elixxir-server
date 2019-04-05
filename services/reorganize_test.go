////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package services

import (
	"testing"
	"gitlab.com/elixxir/crypto/shuffle"
)

type mockSlot struct {
	slot uint64
}

func (g *mockSlot) SlotID() uint64 {
	return g.slot
}

func printListOfSlots(slots []*Slot, t *testing.T) {
	for i := 0; i < len(slots); i++ {
		t.Log((*slots[i]).SlotID())
	}
}

func TestReorganizeSlots(t *testing.T) {

	batchSize := uint64(400)
	testUints := make([]uint64, batchSize)
	shuffledTestSlots := make([]*Slot, batchSize)
	for i := uint64(0); i < batchSize; i++ {
		testUints[i] = uint64(i)
		slot := (Slot)(&mockSlot{slot: uint64(i)})
		shuffledTestSlots[i] = &slot
	}

	for i := 0; i < 100; i++ {
		tc := NewSlotReorganizer(nil, nil, batchSize)

		shuffle.Shuffle(&testUints)
		for j := uint64(0); j < batchSize; j++ {
			tc.InChannel <- shuffledTestSlots[j]
		}
		for j := uint64(0); j < batchSize; j++ {
			shuffledTestSlots[j] = <-tc.OutChannel
		}

		// See if the list is in order
		for j := uint64(1); j < batchSize; j++ {
			if (*shuffledTestSlots[j]).SlotID() <= (*shuffledTestSlots[j-1]).SlotID() {
				t.Errorf("Slice of slots was not in order at index %v\n", j-1)
				printListOfSlots(shuffledTestSlots, t)
			}
		}
	}
}
