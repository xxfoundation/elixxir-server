////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package precomputation

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/services"
	"testing"
)

func TestStrip(t *testing.T) {
	// NOTE: Does not test correctness.

	test := 3
	pass := 0

	batchSize := uint64(3)

	round := globals.NewRound(batchSize)

	rng := cyclic.NewRandom(cyclic.NewInt(2), cyclic.NewInt(2000))

	group := cyclic.NewGroup(cyclic.NewInt(199), cyclic.NewInt(11),
		cyclic.NewInt(13), rng)

	var inMessages []services.Slot

	inMessages = append(inMessages, &PrecomputationSlot{Slot: uint64(0),
		MessagePrecomputation:        cyclic.NewInt(39),
		AssociatedDataPrecomputation: cyclic.NewInt(13)})

	inMessages = append(inMessages, &PrecomputationSlot{Slot: uint64(1),
		MessagePrecomputation:        cyclic.NewInt(86),
		AssociatedDataPrecomputation: cyclic.NewInt(87)})

	inMessages = append(inMessages, &PrecomputationSlot{Slot: uint64(2),
		MessagePrecomputation:        cyclic.NewInt(39),
		AssociatedDataPrecomputation: cyclic.NewInt(51)})

	globals.InitLastNode(round)
	round.LastNode.EncryptedMessagePrecomputation[0] = cyclic.NewInt(41)
	round.LastNode.EncryptedAssociatedDataPrecomputation[0] = cyclic.NewInt(74)
	round.LastNode.EncryptedMessagePrecomputation[1] = cyclic.NewInt(8)
	round.LastNode.EncryptedAssociatedDataPrecomputation[1] = cyclic.NewInt(49)
	round.LastNode.EncryptedMessagePrecomputation[2] = cyclic.NewInt(91)
	round.LastNode.EncryptedAssociatedDataPrecomputation[2] = cyclic.NewInt(73)

	expected := []PrecomputationSlot{
		PrecomputationSlot{Slot: uint64(0),
			MessagePrecomputation:        cyclic.NewInt(98),
			AssociatedDataPrecomputation: cyclic.NewInt(21)},
		PrecomputationSlot{Slot: uint64(1),
			MessagePrecomputation:        cyclic.NewInt(51),
			AssociatedDataPrecomputation: cyclic.NewInt(12)},
		PrecomputationSlot{Slot: uint64(2),
			MessagePrecomputation:        cyclic.NewInt(135),
			AssociatedDataPrecomputation: cyclic.NewInt(138)},
	}

	dc := services.DispatchCryptop(&group, Strip{}, nil, nil, round)

	for i := uint64(0); i < batchSize; i++ {
		dc.InChannel <- &(inMessages[i])
		act := <-dc.OutChannel
		actual := (*act).(*PrecomputationSlot)

		if actual.SlotID() != expected[i].SlotID() {
			t.Errorf("Test of Precomputation Strip's cryptop failed Slot"+
				"ID Test on index: %v; Expected: %v; Actual: %v\n", i,
				expected[i].SlotID(), actual.SlotID())
		} else if actual.MessagePrecomputation.Cmp(
			expected[i].MessagePrecomputation) != 0 {
			t.Errorf("Test of Precomputation Strip's cryptop failed"+
				" MessagePrecomputation "+
				"on index: %v; Expected: %v; Actual: %v\n", i,
				expected[i].MessagePrecomputation.Text(10),
				actual.MessagePrecomputation.Text(10))
		} else if actual.AssociatedDataPrecomputation.Cmp(
			expected[i].AssociatedDataPrecomputation) != 0 {
			t.Errorf("Test of Precomputation Strip's cryptop failed"+
				" AssociatedDataPrecomputation "+
				"on index: %v; Expected: %v; Actual: %v\n", i,
				expected[i].AssociatedDataPrecomputation.Text(10),
				actual.AssociatedDataPrecomputation.Text(10))
		} else {
			pass++
		}
	}

	println("Precomputation Strip", pass, "out of", test, "tests "+
		"passed.")
}
