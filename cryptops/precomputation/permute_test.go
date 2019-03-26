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

func TestPermute(t *testing.T) {
	// NOTE: Does not test correctness.

	test := 3
	pass := 0

	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(23),
		large.NewInt(27))

	batchSize := uint64(3)

	round := globals.NewRound(batchSize, &grp)

	round.Permutations[0] = 1
	round.Permutations[1] = 2
	round.Permutations[2] = 0

	round.Z = grp.NewInt(30)
	//globals.Grp.G = grp.NewInt(55)

	round.S_INV[0] = grp.NewInt(53)
	round.S_INV[1] = grp.NewInt(24)
	round.S_INV[2] = grp.NewInt(61)

	round.V_INV[0] = grp.NewInt(52)
	round.V_INV[1] = grp.NewInt(68)
	round.V_INV[2] = grp.NewInt(11)

	round.Y_S[0] = grp.NewInt(98)
	round.Y_S[1] = grp.NewInt(7)
	round.Y_S[2] = grp.NewInt(32)

	round.Y_V[0] = grp.NewInt(23)
	round.Y_V[1] = grp.NewInt(16)
	round.Y_V[2] = grp.NewInt(17)

	var inMessages []services.Slot
	inMessages = append(inMessages, &PrecomputationSlot{Slot: uint64(0),
		MessageCypher:                grp.NewInt(39),
		AssociatedDataCypher:         grp.NewInt(13),
		MessagePrecomputation:        grp.NewInt(41),
		AssociatedDataPrecomputation: grp.NewInt(74)})

	inMessages = append(inMessages, &PrecomputationSlot{Slot: uint64(1),
		MessageCypher:                grp.NewInt(86),
		AssociatedDataCypher:         grp.NewInt(87),
		MessagePrecomputation:        grp.NewInt(8),
		AssociatedDataPrecomputation: grp.NewInt(49)})

	inMessages = append(inMessages, &PrecomputationSlot{Slot: uint64(2),
		MessageCypher:                grp.NewInt(39),
		AssociatedDataCypher:         grp.NewInt(51),
		MessagePrecomputation:        grp.NewInt(91),
		AssociatedDataPrecomputation: grp.NewInt(73)})

	expected := []PrecomputationSlot{
		PrecomputationSlot{
			Slot:                         uint64(1),
			MessageCypher:                grp.NewInt(56),
			AssociatedDataCypher:         grp.NewInt(35),
			MessagePrecomputation:        grp.NewInt(56),
			AssociatedDataPrecomputation: grp.NewInt(89),
		},
		PrecomputationSlot{
			Slot:                         uint64(2),
			MessageCypher:                grp.NewInt(60),
			AssociatedDataCypher:         grp.NewInt(97),
			MessagePrecomputation:        grp.NewInt(92),
			AssociatedDataPrecomputation: grp.NewInt(48),
		},
		PrecomputationSlot{
			Slot:                         uint64(0),
			MessageCypher:                grp.NewInt(34),
			AssociatedDataCypher:         grp.NewInt(98),
			MessagePrecomputation:        grp.NewInt(58),
			AssociatedDataPrecomputation: grp.NewInt(16),
		},
	}
	dispatch := services.DispatchCryptop(&grp, Permute{}, nil, nil, round)

	for i := uint64(0); i < batchSize; i++ {
		dispatch.InChannel <- &(inMessages[i])
		actual := <-dispatch.OutChannel
		act := (*actual).(*PrecomputationSlot)

		if act.SlotID() != expected[i].SlotID() {
			t.Errorf("Test of Precomputation Permute's cryptop failed Slot"+
				"ID Test on index: %v; Expected: %v; Actual: %v\n", i,
				expected[i].SlotID(), act.SlotID())
		} else if act.MessageCypher.Cmp(expected[i].MessageCypher) != 0 {
			t.Errorf("Test of Precomputation Permute's cryptop failed Message"+
				"Keys Test on index: %v\n\tExpected: %#v\n\tActual:   %#v", i,
				expected[i].MessageCypher.Text(10),
				act.MessageCypher.Text(10))
		} else if act.AssociatedDataCypher.Cmp(expected[i].AssociatedDataCypher) != 0 {
			t.Errorf("Test of Precomputation Permute's cryptop failed Recipient"+
				"IDKeys Test on index: %v\n\tExpected: %#v\n\tActual:   %#v", i,
				expected[i].AssociatedDataCypher.Text(10),
				act.AssociatedDataCypher.Text(10))
		} else if act.MessagePrecomputation.Cmp(expected[i].MessagePrecomputation) != 0 {
			t.Errorf("Test of Precomputation Permute's cryptop failed Message"+
				"CypherText Test on index: %v\n\tExpected: %#v\n\tActual:   %#v", i,
				expected[i].MessagePrecomputation.Text(10),
				act.MessagePrecomputation.Text(10))
		} else if act.AssociatedDataPrecomputation.Cmp(expected[i].AssociatedDataPrecomputation) != 0 {
			t.Errorf("Test of Precomputation Permute's cryptop failed Recipient"+
				"CypherText Test on index: %v\n\tExpected: %#v\n\tActual:   %#v", i,
				expected[i].AssociatedDataPrecomputation.Text(10),
				act.AssociatedDataPrecomputation.Text(10))
		} else {
			pass++
		}
	}

	println("Precomputation Permute", pass, "out of", test, "tests "+
		"passed.")
}
