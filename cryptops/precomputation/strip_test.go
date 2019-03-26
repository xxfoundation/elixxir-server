////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package precomputation

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/services"
	"testing"
)

func TestStrip(t *testing.T) {
	// NOTE: Does not test correctness.

	test := 3
	pass := 0

	grp := cyclic.NewGroup(large.NewInt(199), large.NewInt(11),
		large.NewInt(13))

	batchSize := uint64(3)

	round := globals.NewRound(batchSize, &grp)

	var inMessages []services.Slot

	inMessages = append(inMessages, &PrecomputationSlot{Slot: uint64(0),
		MessagePrecomputation:        grp.NewInt(39),
		AssociatedDataPrecomputation: grp.NewInt(13)})

	inMessages = append(inMessages, &PrecomputationSlot{Slot: uint64(1),
		MessagePrecomputation:        grp.NewInt(86),
		AssociatedDataPrecomputation: grp.NewInt(87)})

	inMessages = append(inMessages, &PrecomputationSlot{Slot: uint64(2),
		MessagePrecomputation:        grp.NewInt(39),
		AssociatedDataPrecomputation: grp.NewInt(51)})

	globals.InitLastNode(round, &grp)
	round.LastNode.EncryptedMessagePrecomputation[0] = grp.NewInt(41)
	round.LastNode.EncryptedAssociatedDataPrecomputation[0] = grp.NewInt(74)
	round.LastNode.EncryptedMessagePrecomputation[1] = grp.NewInt(8)
	round.LastNode.EncryptedAssociatedDataPrecomputation[1] = grp.NewInt(49)
	round.LastNode.EncryptedMessagePrecomputation[2] = grp.NewInt(91)
	round.LastNode.EncryptedAssociatedDataPrecomputation[2] = grp.NewInt(73)

	expected := []PrecomputationSlot{
		PrecomputationSlot{Slot: uint64(0),
			MessagePrecomputation:        grp.NewInt(98),
			AssociatedDataPrecomputation: grp.NewInt(21)},
		PrecomputationSlot{Slot: uint64(1),
			MessagePrecomputation:        grp.NewInt(51),
			AssociatedDataPrecomputation: grp.NewInt(12)},
		PrecomputationSlot{Slot: uint64(2),
			MessagePrecomputation:        grp.NewInt(135),
			AssociatedDataPrecomputation: grp.NewInt(138)},
	}

	dc := services.DispatchCryptop(&grp, Strip{}, nil, nil, round)

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
