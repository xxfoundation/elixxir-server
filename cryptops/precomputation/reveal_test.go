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

func TestPrecomputationReveal(t *testing.T) {

	test := 3
	pass := 0

	var im []services.Slot

	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(23),
		large.NewInt(29))

	bs := uint64(3)

	round := globals.NewRound(bs, &grp)

	im = append(im, &PrecomputationSlot{
		Slot:                         uint64(0),
		MessagePrecomputation:        grp.NewInt(int64(39)),
		AssociatedDataPrecomputation: grp.NewInt(int64(13))})

	im = append(im, &PrecomputationSlot{
		Slot:                         uint64(1),
		MessagePrecomputation:        grp.NewInt(int64(86)),
		AssociatedDataPrecomputation: grp.NewInt(int64(87))})

	im = append(im, &PrecomputationSlot{
		Slot:                         uint64(2),
		MessagePrecomputation:        grp.NewInt(int64(39)),
		AssociatedDataPrecomputation: grp.NewInt(int64(51))})

	round.Z = grp.NewInt(53)

	results := [][]*cyclic.Int{
		{grp.NewInt(53), grp.NewInt(14)},
		{grp.NewInt(11), grp.NewInt(10)},
		{grp.NewInt(53), grp.NewInt(68)},
	}

	dc := services.DispatchCryptop(&grp, Reveal{}, nil, nil, round)

	for i := uint64(0); i < bs; i++ {
		dc.InChannel <- &im[i]
		trn := <-dc.OutChannel

		rtn := (*trn).(*PrecomputationSlot)
		result := results[i]

		if result[0].Cmp(rtn.MessagePrecomputation) != 0 ||
			result[1].Cmp(rtn.AssociatedDataPrecomputation) != 0 {
			t.Errorf("Test of PrecompReveal's cryptop failed on index: %v"+
				" Expected: %v,%v Received: %v,%v ", i,
				result[0].Text(10), result[1].Text(10),
				rtn.MessagePrecomputation.Text(10),
				rtn.AssociatedDataPrecomputation.Text(10))
		} else {
			pass++
		}
	}

	println("PrecompReveal", pass, "out of", test, "tests passed.")

}

func TestPrecomputationRevealRun(t *testing.T) {
	bs := uint64(3)

	var im []*PrecomputationSlot
	var om []*PrecomputationSlot

	grp := cyclic.NewGroup(large.NewInt(101), large.NewInt(23),
		large.NewInt(29))

	im = append(im, &PrecomputationSlot{
		Slot:                         uint64(0),
		MessagePrecomputation:        grp.NewInt(int64(39)),
		AssociatedDataPrecomputation: grp.NewInt(int64(13))})

	im = append(im, &PrecomputationSlot{
		Slot:                         uint64(1),
		MessagePrecomputation:        grp.NewInt(int64(86)),
		AssociatedDataPrecomputation: grp.NewInt(int64(87))})

	im = append(im, &PrecomputationSlot{
		Slot:                         uint64(2),
		MessagePrecomputation:        grp.NewInt(int64(39)),
		AssociatedDataPrecomputation: grp.NewInt(int64(51))})

	om = append(om, &PrecomputationSlot{
		Slot:                         uint64(1),
		MessagePrecomputation:        grp.NewInt(int64(0)),
		AssociatedDataPrecomputation: grp.NewInt(int64(0))})

	om = append(om, &PrecomputationSlot{
		Slot:                         uint64(2),
		MessagePrecomputation:        grp.NewInt(int64(0)),
		AssociatedDataPrecomputation: grp.NewInt(int64(0))})

	om = append(om, &PrecomputationSlot{
		Slot:                         uint64(0),
		MessagePrecomputation:        grp.NewInt(int64(0)),
		AssociatedDataPrecomputation: grp.NewInt(int64(0))})

	key := KeysReveal{
		Z: grp.NewInt(53),
	}

	results := [][]*cyclic.Int{
		{grp.NewInt(60), grp.NewInt(77)},
		{grp.NewInt(34), grp.NewInt(95)},
		{grp.NewInt(60), grp.NewInt(66)},
	}

	reveal := Reveal{}

	for i := uint64(0); i < bs; i++ {
		reveal.Run(&grp, im[i], om[i], &key)
	}

	for i := uint64(0); i < bs; i++ {
		if results[i][0].Cmp(om[i].MessagePrecomputation) != 0 ||
			results[i][1].Cmp(om[i].AssociatedDataPrecomputation) != 0 {
			t.Errorf("TestPrecomputationRevealRun - Expected: %v,%v Got: %v,%v",
				results[i][0].Text(10), results[i][1].Text(10),
				om[i].MessagePrecomputation.Text(10),
				om[i].AssociatedDataPrecomputation.Text(10))
		}
	}
}
