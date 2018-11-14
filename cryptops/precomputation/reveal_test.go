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

func TestPrecomputationReveal(t *testing.T) {

	test := 6
	pass := 0

	bs := uint64(3)

	round := globals.NewRound(bs)

	var im []services.Slot

	rng := cyclic.NewRandom(cyclic.NewInt(0), cyclic.NewInt(1000))

	grp := cyclic.NewGroup(cyclic.NewInt(101), cyclic.NewInt(23),
		cyclic.NewInt(29), rng)

	im = append(im, &PrecomputationSlot{
		Slot: uint64(0),
		MessagePrecomputation:     cyclic.NewInt(int64(39)),
		RecipientIDPrecomputation: cyclic.NewInt(int64(13))})

	im = append(im, &PrecomputationSlot{
		Slot: uint64(1),
		MessagePrecomputation:     cyclic.NewInt(int64(86)),
		RecipientIDPrecomputation: cyclic.NewInt(int64(87))})

	im = append(im, &PrecomputationSlot{
		Slot: uint64(2),
		MessagePrecomputation:     cyclic.NewInt(int64(39)),
		RecipientIDPrecomputation: cyclic.NewInt(int64(51))})

	round.Z = cyclic.NewInt(53)

	results := [][]*cyclic.Int{
		{cyclic.NewInt(60), cyclic.NewInt(77)},
		{cyclic.NewInt(34), cyclic.NewInt(95)},
		{cyclic.NewInt(60), cyclic.NewInt(66)},
	}

	dc := services.DispatchCryptop(&grp, Reveal{}, nil, nil, round)

	for i := uint64(0); i < bs; i++ {
		dc.InChannel <- &im[i]
		trn := <-dc.OutChannel

		rtn := (*trn).(*PrecomputationSlot)
		result := results[i]

		if result[0].Cmp(rtn.MessagePrecomputation) != 0 ||
			result[1].Cmp(rtn.RecipientIDPrecomputation) != 0 {
			t.Errorf("Test of PrecompReveal's cryptop failed on index: %v"+
				" Expected: %v,%v Received: %v,%v ", i,
				result[0].Text(10), result[1].Text(10),
				rtn.MessagePrecomputation.Text(10),
				rtn.RecipientIDPrecomputation.Text(10))
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

	rng := cyclic.NewRandom(cyclic.NewInt(0), cyclic.NewInt(1000))

	grp := cyclic.NewGroup(cyclic.NewInt(101), cyclic.NewInt(23),
		cyclic.NewInt(29), rng)

	im = append(im, &PrecomputationSlot{
		Slot: uint64(0),
		MessagePrecomputation:     cyclic.NewInt(int64(39)),
		RecipientIDPrecomputation: cyclic.NewInt(int64(13))})

	im = append(im, &PrecomputationSlot{
		Slot: uint64(1),
		MessagePrecomputation:     cyclic.NewInt(int64(86)),
		RecipientIDPrecomputation: cyclic.NewInt(int64(87))})

	im = append(im, &PrecomputationSlot{
		Slot: uint64(2),
		MessagePrecomputation:     cyclic.NewInt(int64(39)),
		RecipientIDPrecomputation: cyclic.NewInt(int64(51))})

	om = append(om, &PrecomputationSlot{
		Slot: uint64(1),
		MessagePrecomputation:     cyclic.NewInt(int64(0)),
		RecipientIDPrecomputation: cyclic.NewInt(int64(0))})

	om = append(om, &PrecomputationSlot{
		Slot: uint64(2),
		MessagePrecomputation:     cyclic.NewInt(int64(0)),
		RecipientIDPrecomputation: cyclic.NewInt(int64(0))})

	om = append(om, &PrecomputationSlot{
		Slot: uint64(0),
		MessagePrecomputation:     cyclic.NewInt(int64(0)),
		RecipientIDPrecomputation: cyclic.NewInt(int64(0))})

	key := KeysReveal{
		Z: cyclic.NewInt(53),
	}

	results := [][]*cyclic.Int{
		{cyclic.NewInt(60), cyclic.NewInt(77)},
		{cyclic.NewInt(34), cyclic.NewInt(95)},
		{cyclic.NewInt(60), cyclic.NewInt(66)},
	}

	reveal := Reveal{}

	for i := uint64(0); i < bs; i++ {
		reveal.Run(&grp, im[i], om[i], &key)
	}

	for i := uint64(0); i < bs; i++ {
		if results[i][0].Cmp(om[i].MessagePrecomputation) != 0 ||
			results[i][1].Cmp(om[i].RecipientIDPrecomputation) != 0 {
			t.Errorf("TestPrecomputationRevealRun - Expected: %v,%v Got: %v,%v",
				results[i][0].Text(10), results[i][1].Text(10),
				om[i].MessagePrecomputation.Text(10),
				om[i].RecipientIDPrecomputation.Text(10))
		}
	}
}
