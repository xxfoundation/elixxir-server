////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package realtime

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/services"
	"testing"
)

func TestPeel(t *testing.T) {
	// NOTE: Does not test correctness

	test := 6
	pass := 0

	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(27),
		large.NewInt(97))

	bs := uint64(3)

	round := globals.NewRound(bs, &grp)

	recipientIds := [3]*id.User{
		id.NewUserFromUint(5, t),
		id.NewUserFromUint(7, t),
		id.NewUserFromUint(9, t),
	}

	var im []services.Slot

	im = append(im, &Slot{
		Slot:      uint64(0),
		CurrentID: recipientIds[0],
		Message:   grp.NewInt(int64(39))})

	im = append(im, &Slot{
		Slot:      uint64(1),
		CurrentID: recipientIds[1],
		Message:   grp.NewInt(int64(86))})

	im = append(im, &Slot{
		Slot:      uint64(2),
		CurrentID: recipientIds[2],
		Message:   grp.NewInt(int64(66))})

	// Set the keys
	round.LastNode.MessagePrecomputation = make([]*cyclic.Int, round.BatchSize)
	round.LastNode.MessagePrecomputation[0] = grp.NewInt(77)
	round.LastNode.MessagePrecomputation[1] = grp.NewInt(93)
	round.LastNode.MessagePrecomputation[2] = grp.NewInt(47)

	expected := [][]*cyclic.Int{
		{grp.NewInt(7)},
		{grp.NewInt(80)},
		{grp.NewInt(106)},
	}

	dc := services.DispatchCryptop(&grp, Peel{}, nil, nil, round)

	for i := uint64(0); i < bs; i++ {
		dc.InChannel <- &(im[i])
		rtn := <-dc.OutChannel

		result := expected[i]

		rtnXtc := (*rtn).(*Slot)

		// Test EncryptedMessage results
		for j := 0; j < 1; j++ {
			if result[j].Cmp(rtnXtc.Message) != 0 {
				t.Errorf("Test of RealtimePeel's EncryptedMessage output "+
					"failed on index: %v on value: %v.  Expected: %v Received: %v ",
					i, j, result[j].Text(10), rtnXtc.Message.Text(10))
			} else {
				pass++
			}
		}

		// Test AssociatedData pass through
		if recipientIds[i] != rtnXtc.CurrentID {
			t.Errorf("Test of RealtimePeel's AssociatedData ouput failed on index %v.  Expected: %v Received: %v ",
				i, recipientIds[i], rtnXtc.CurrentID)
		} else {
			pass++
		}
	}

	println("Realtime Peel", pass, "out of", test, "tests passed.")

}
