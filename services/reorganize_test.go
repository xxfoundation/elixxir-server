package services

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"testing"
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

	tests := 100
	pass := 0

	for i := 0; i < tests; i++ {
		tc := NewSlotReorganizer(nil, nil, batchSize)

		cyclic.Shuffle(&testUints)
		for i := uint64(0); i < batchSize; i++ {
			tc.InChannel <- shuffledTestSlots[i]
		}
		for i := uint64(0); i < batchSize; i++ {
			shuffledTestSlots[i] = <-tc.OutChannel
		}

		// See if the list is in order
		for i := uint64(1); i < batchSize; i++ {
			if (*shuffledTestSlots[i]).SlotID() <= (*shuffledTestSlots[i-1]).SlotID() {
				t.Errorf("Slice of slots was not in order at index %v\n", i-1)
				printListOfSlots(shuffledTestSlots, t)
			}
		}

		if !t.Failed() {
			pass++
		}
	}

	println("Reorganize Slots:", pass, "out of", tests, "passed.")

	tc.Kill(false)
}
