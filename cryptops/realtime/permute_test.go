////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package realtime

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/services"
	"testing"
)

func TestRealTimePermute(t *testing.T) {

	test := 6
	pass := 0

	var im []services.Slot

	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(23),
		large.NewInt(29))

	bs := uint64(3)

	round := globals.NewRound(bs, grp)

	im = append(im, &Slot{
		Slot:           uint64(0),
		Message:        grp.NewInt(int64(39)),
		AssociatedData: grp.NewInt(int64(13))})

	im = append(im, &Slot{
		Slot:           uint64(1),
		Message:        grp.NewInt(int64(86)),
		AssociatedData: grp.NewInt(int64(87))})

	im = append(im, &Slot{
		Slot:           uint64(2),
		Message:        grp.NewInt(int64(39)),
		AssociatedData: grp.NewInt(int64(51))})

	round.Permutations[0] = 1
	round.Permutations[1] = 2
	round.Permutations[2] = 0

	round.S[0] = grp.NewInt(53)
	round.S[1] = grp.NewInt(24)
	round.S[2] = grp.NewInt(61)

	round.V[0] = grp.NewInt(52)
	round.V[1] = grp.NewInt(68)
	round.V[2] = grp.NewInt(11)

	results := [][]*cyclic.Int{
		{grp.NewInt(34), grp.NewInt(34)},
		{grp.NewInt(31), grp.NewInt(31)},
		{grp.NewInt(25), grp.NewInt(26)},
	}

	dc := services.DispatchCryptop(grp, Permute{}, nil, nil, round)

	for i := uint64(0); i < bs; i++ {
		dc.InChannel <- &im[i]
		trn := <-dc.OutChannel

		rtn := (*trn).(*Slot)
		result := results[i]

		if result[0].Cmp(rtn.Message) != 0 ||
			result[1].Cmp(rtn.AssociatedData) != 0 {
			t.Errorf("Test of RealPermute's cryptop failed on index: %v"+
				" Expected: %v,%v Received: %v,%v ", i,
				result[0].Text(10), result[1].Text(10),
				rtn.Message.Text(10), rtn.AssociatedData.Text(10))
		} else {
			pass++
		}

		if rtn.SlotID() == i {
			t.Errorf("Test of RealPermute's permute failed on index: %v", i)
		} else {
			pass++
		}

	}

	println("RealPermute", pass, "out of", test, "tests passed.")

}

func TestRealtimePermuteRun(t *testing.T) {
	bs := uint64(3)

	var im []*Slot
	var om []*Slot

	grp := cyclic.NewGroup(large.NewInt(101), large.NewInt(23),
		large.NewInt(29))

	im = append(im, &Slot{
		Slot:           uint64(0),
		Message:        grp.NewInt(int64(39)),
		AssociatedData: grp.NewInt(int64(13))})

	im = append(im, &Slot{
		Slot:           uint64(1),
		Message:        grp.NewInt(int64(86)),
		AssociatedData: grp.NewInt(int64(87))})

	im = append(im, &Slot{
		Slot:           uint64(2),
		Message:        grp.NewInt(int64(39)),
		AssociatedData: grp.NewInt(int64(51))})

	om = append(om, &Slot{
		Slot:           uint64(1),
		Message:        grp.NewInt(int64(1)),
		AssociatedData: grp.NewInt(int64(1))})

	om = append(om, &Slot{
		Slot:           uint64(2),
		Message:        grp.NewInt(int64(1)),
		AssociatedData: grp.NewInt(int64(1))})

	om = append(om, &Slot{
		Slot:           uint64(0),
		Message:        grp.NewInt(int64(1)),
		AssociatedData: grp.NewInt(int64(1))})

	keys := []KeysPermute{
		{
			S: grp.NewInt(53),
			V: grp.NewInt(52)},
		{
			S: grp.NewInt(24),
			V: grp.NewInt(68)},
		{
			S: grp.NewInt(61),
			V: grp.NewInt(11)},
	}

	results := [][]*cyclic.Int{
		{grp.NewInt(47), grp.NewInt(70)},
		{grp.NewInt(44), grp.NewInt(58)},
		{grp.NewInt(56), grp.NewInt(56)},
	}

	permute := Permute{}

	for i := uint64(0); i < bs; i++ {
		permute.Run(grp, im[i], om[i], &keys[i])
	}

	for i := uint64(0); i < bs; i++ {
		if results[i][0].Cmp(om[i].Message) != 0 ||
			results[i][1].Cmp(om[i].AssociatedData) != 0 {
			t.Errorf("TestRealtimePermuteRun - Expected: %v,%v Got: %v,%v",
				results[i][0].Text(10), results[i][1].Text(10),
				om[i].Message.Text(10),
				om[i].AssociatedData.Text(10))
		}
	}
}
