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

func TestPermute(t *testing.T) {
	// NOTE: Does not test correctness.

	test := 3
	pass := 0

	batchSize := uint64(3)

	round := globals.NewRound(batchSize)

	rng := cyclic.NewRandom(cyclic.NewInt(0), cyclic.NewInt(1000))

	group := cyclic.NewGroup(cyclic.NewInt(101), cyclic.NewInt(23),
		cyclic.NewInt(27), rng)

	round.Permutations[0] = 1
	round.Permutations[1] = 2
	round.Permutations[2] = 0

	round.Z = cyclic.NewInt(30)
	globals.Grp.G = cyclic.NewInt(55)

	round.S_INV[0] = cyclic.NewInt(53)
	round.S_INV[1] = cyclic.NewInt(24)
	round.S_INV[2] = cyclic.NewInt(61)

	round.V_INV[0] = cyclic.NewInt(52)
	round.V_INV[1] = cyclic.NewInt(68)
	round.V_INV[2] = cyclic.NewInt(11)

	round.Y_S[0] = cyclic.NewInt(98)
	round.Y_S[1] = cyclic.NewInt(7)
	round.Y_S[2] = cyclic.NewInt(32)

	round.Y_V[0] = cyclic.NewInt(23)
	round.Y_V[1] = cyclic.NewInt(16)
	round.Y_V[2] = cyclic.NewInt(17)

	var inMessages []services.Slot
	inMessages = append(inMessages, &PrecomputationSlot{Slot: uint64(0),
		MessageCypher:             cyclic.NewInt(39),
		AssociatedDataCypher:         cyclic.NewInt(13),
		MessagePrecomputation:     cyclic.NewInt(41),
		AssociatedDataPrecomputation: cyclic.NewInt(74)})

	inMessages = append(inMessages, &PrecomputationSlot{Slot: uint64(1),
		MessageCypher:             cyclic.NewInt(86),
		AssociatedDataCypher:         cyclic.NewInt(87),
		MessagePrecomputation:     cyclic.NewInt(8),
		AssociatedDataPrecomputation: cyclic.NewInt(49)})

	inMessages = append(inMessages, &PrecomputationSlot{Slot: uint64(2),
		MessageCypher:             cyclic.NewInt(39),
		AssociatedDataCypher:         cyclic.NewInt(51),
		MessagePrecomputation:     cyclic.NewInt(91),
		AssociatedDataPrecomputation: cyclic.NewInt(73)})

	expected := []PrecomputationSlot{
		PrecomputationSlot{Slot: uint64(1),
			MessageCypher:             cyclic.NewInt(71),
			AssociatedDataCypher:         cyclic.NewInt(60),
			MessagePrecomputation:     cyclic.NewInt(44),
			AssociatedDataPrecomputation: cyclic.NewInt(97)},
		PrecomputationSlot{Slot: uint64(2),
			MessageCypher:             cyclic.NewInt(79),
			AssociatedDataCypher:         cyclic.NewInt(16),
			MessagePrecomputation:     cyclic.NewInt(47),
			AssociatedDataPrecomputation: cyclic.NewInt(47)},
		PrecomputationSlot{Slot: uint64(0),
			MessageCypher:             cyclic.NewInt(78),
			AssociatedDataCypher:         cyclic.NewInt(34),
			MessagePrecomputation:     cyclic.NewInt(69),
			AssociatedDataPrecomputation: cyclic.NewInt(13)},
	}
	dispatch := services.DispatchCryptop(&group, Permute{}, nil, nil, round)

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
				"Keys Test on index: %v; Expected: %v; Actual: %v\n", i,
				expected[i].MessageCypher.Text(10),
				act.MessageCypher.Text(10))
		} else if act.AssociatedDataCypher.Cmp(expected[i].AssociatedDataCypher) != 0 {
			t.Errorf("Test of Precomputation Permute's cryptop failed Recipient"+
				"IDKeys Test on index: %v; Expected: %v; Actual: %v\n", i,
				expected[i].AssociatedDataCypher.Text(10),
				act.AssociatedDataCypher.Text(10))
		} else if act.MessagePrecomputation.Cmp(expected[i].MessagePrecomputation) != 0 {
			t.Errorf("Test of Precomputation Permute's cryptop failed Message"+
				"CypherText Test on index: %v; Expected: %v; Actual: %v\n", i,
				expected[i].MessagePrecomputation.Text(10),
				act.MessagePrecomputation.Text(10))
		} else if act.AssociatedDataPrecomputation.Cmp(expected[i].AssociatedDataPrecomputation) != 0 {
			t.Errorf("Test of Precomputation Permute's cryptop failed Recipient"+
				"CypherText Test on index: %v; Expected: %v; Actual: %v\n", i,
				expected[i].AssociatedDataPrecomputation.Text(10),
				act.AssociatedDataPrecomputation.Text(10))
		} else {
			pass++
		}
	}

	println("Precomputation Permute", pass, "out of", test, "tests "+
		"passed.")
}
