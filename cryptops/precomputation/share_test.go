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

func TestShare(t *testing.T) {
	// NOTE: Does not test correctness

	test := 3
	pass := 0

	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(23),
		large.NewInt(27))

	bs := uint64(3)

	round := globals.NewRound(bs, &grp)

	var im []services.Slot

	im = append(im, &SlotShare{
		Slot:                        uint64(0),
		PartialRoundPublicCypherKey: grp.NewInt(int64(39))})

	im = append(im, &SlotShare{
		Slot:                        uint64(1),
		PartialRoundPublicCypherKey: grp.NewInt(int64(86))})

	im = append(im, &SlotShare{
		Slot:                        uint64(1),
		PartialRoundPublicCypherKey: grp.NewInt(int64(66))})

	round.Z = grp.NewInt(53)

	expected := [][]*cyclic.Int{
		{grp.NewInt(1)},
		{grp.NewInt(1)},
		{grp.NewInt(106)},
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
