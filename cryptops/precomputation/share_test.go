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

func TestShare(t *testing.T) {
	// NOTE: Does not test correctness

	test := 3
	pass := 0

	bs := uint64(3)

	round := globals.NewRound(bs)

	rng := cyclic.NewRandom(cyclic.NewInt(0), cyclic.NewInt(1000))

	grp := cyclic.NewGroup(cyclic.NewInt(101), cyclic.NewInt(23), cyclic.NewInt(27), rng)

	var im []services.Slot

	im = append(im, &SlotShare{
		Slot: uint64(0),
		PartialRoundPublicCypherKey: cyclic.NewInt(int64(39))})

	im = append(im, &SlotShare{
		Slot: uint64(1),
		PartialRoundPublicCypherKey: cyclic.NewInt(int64(86))})

	im = append(im, &SlotShare{
		Slot: uint64(1),
		PartialRoundPublicCypherKey: cyclic.NewInt(int64(66))})

	round.Z = cyclic.NewInt(53)

	expected := [][]*cyclic.Int{
		{cyclic.NewInt(69)},
		{cyclic.NewInt(42)},
		{cyclic.NewInt(51)},
	}

	dc := services.DispatchCryptop(&grp, Share{}, nil, nil, round)

	for i := uint64(0); i < bs; i++ {
		dc.InChannel <- &(im[i])
		rtn := <-dc.OutChannel

		result := expected[i]

		rtnXtc := (*rtn).(*SlotShare)

		for j := 0; j < 1; j++ {
			if result[j].Cmp(rtnXtc.PartialRoundPublicCypherKey) != 0 {
				t.Errorf("Test of PrecompShare's cryptop failed on index: %v on value: %v.  Expected: %v Received: %v ",
					i, j, result[j].Text(10), rtnXtc.PartialRoundPublicCypherKey.Text(10))
			} else {
				pass++
			}
		}

	}

	println("Precomputation Share", pass, "out of", test, "tests passed.")

}
