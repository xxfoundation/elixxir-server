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

func TestEncrypt(t *testing.T) {
	// NOTE: Does not test correctness
	var im []services.Slot

	grp := cyclic.NewGroup(
		large.NewInt(107), large.NewInt(55), large.NewInt(23))

	globals.Clear(t)
	globals.SetGroup(grp)

	batchSize := uint64(3)

	round := globals.NewRound(batchSize, grp)

	im = append(im, &PrecomputationSlot{
		Slot:                  uint64(0),
		MessageCypher:         grp.NewInt(int64(91)),
		MessagePrecomputation: grp.NewInt(int64(73)),
	})

	im = append(im, &PrecomputationSlot{
		Slot:                  uint64(1),
		MessageCypher:         grp.NewInt(int64(86)),
		MessagePrecomputation: grp.NewInt(int64(87)),
	})

	im = append(im, &PrecomputationSlot{
		Slot:                  uint64(2),
		MessageCypher:         grp.NewInt(int64(39)),
		MessagePrecomputation: grp.NewInt(int64(50)),
	})

	round.CypherPublicKey = grp.NewInt(30)

	round.Y_T[0] = grp.NewInt(53)
	round.Y_T[1] = grp.NewInt(24)
	round.Y_T[2] = grp.NewInt(61)

	round.T_INV[0] = grp.NewInt(52)
	round.T_INV[1] = grp.NewInt(68)
	round.T_INV[2] = grp.NewInt(11)

	expected := [][]*cyclic.Int{
		{grp.NewInt(83), grp.NewInt(73)},
		{grp.NewInt(80), grp.NewInt(12)},
		{grp.NewInt(96), grp.NewInt(78)},
	}

	dc := services.DispatchCryptop(
		grp, Encrypt{}, nil, nil, round)

	for i := uint64(0); i < batchSize; i++ {
		dc.InChannel <- &(im[i])
		actual := <-dc.OutChannel

		act := (*actual).(*PrecomputationSlot)

		expectedVal := expected[i]

		if expectedVal[0].Cmp(act.MessageCypher) != 0 {
			t.Errorf("Test of Precomputation Encrypt's cryptop failed Keys Test on index: %v"+
				"\n\tExpected: %#v\n\tActual:   %#v", i, expectedVal[0].Text(10),
				act.MessageCypher.Text(10))
		}
		if expectedVal[1].Cmp(act.MessagePrecomputation) != 0 {
			t.Errorf("Test of Precomputation Encrypt's cryptop failed Cypher Text Test on index: %v"+
				"\n\tExpected: %#v\n\tActual:   %#v", i, expectedVal[1].Text(10),
				act.MessagePrecomputation.Text(10))
		}
	}
}
