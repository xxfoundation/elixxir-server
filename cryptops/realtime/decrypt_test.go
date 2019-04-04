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

func TestDecrypt(t *testing.T) {
	// NOTE: Does not test correctness

	test := 9
	pass := 0

	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(23),
		large.NewInt(27))

	batchSize := uint64(3)

	round := globals.NewRound(batchSize, grp)

	senderIds := [3]*id.User{id.NewUserFromUint(5, t),
		id.NewUserFromUint(7, t),
		id.NewUserFromUint(9, t),
	}

	var im []services.Slot

	im = append(im, &Slot{
		Slot:           uint64(0),
		CurrentID:      senderIds[0],
		Message:        grp.NewInt(int64(39)),
		CurrentKey:     grp.NewInt(int64(65)),
		AssociatedData: grp.NewInt(7)})

	im = append(im, &Slot{
		Slot:           uint64(1),
		CurrentID:      senderIds[1],
		Message:        grp.NewInt(int64(86)),
		CurrentKey:     grp.NewInt(int64(44)),
		AssociatedData: grp.NewInt(51)})

	im = append(im, &Slot{
		Slot:           uint64(2),
		CurrentID:      senderIds[2],
		Message:        grp.NewInt(int64(66)),
		CurrentKey:     grp.NewInt(int64(94)),
		AssociatedData: grp.NewInt(23)})

	// Set the keys
	round.R[0] = grp.NewInt(52)
	round.R[1] = grp.NewInt(68)
	round.R[2] = grp.NewInt(11)

	round.U[0] = grp.NewInt(67)
	round.U[1] = grp.NewInt(88)
	round.U[2] = grp.NewInt(20)

	expected := [][]*cyclic.Int{
		{grp.NewInt(103), grp.NewInt(97)},
		{grp.NewInt(84), grp.NewInt(57)},
		{grp.NewInt(85), grp.NewInt(12)},
	}

	dc := services.DispatchCryptop(grp, Decrypt{}, nil, nil, round)

	for i := uint64(0); i < batchSize; i++ {
		dc.InChannel <- &(im[i])
		rtn := <-dc.OutChannel

		result := expected[i]

		rtnXtc := (*rtn).(*Slot)

		for j := 0; j < 1; j++ {
			// Test EncryptedMessage results
			if result[j].Cmp(rtnXtc.Message) != 0 {
				t.Errorf("Test of RealtimeDecrypt's EncryptedMessage output "+
					"failed on index: %v on value: %v.  Expected: %v Received: %v ",
					i, j, result[j].Text(10), rtnXtc.Message.Text(10))
			} else {
				pass++
			}
			// Test AssociatedData results
			if result[j+1].Cmp(rtnXtc.AssociatedData) != 0 {
				t.Errorf("Test of RealtimeDecrypt's AssociatedData output "+
					"failed on index: %v on value: %v.  Expected: %v Received: %v ",
					i, j+1, result[j+1].Text(10), rtnXtc.AssociatedData.Text(10))
			} else {
				pass++
			}
		}

		// Test SenderID pass through
		if senderIds[i] != rtnXtc.CurrentID {
			t.Errorf("Test of RealtimeDecrypt's SenderID ouput failed on index %v.  Expected: %v Received: %v ",
				i, senderIds[i], rtnXtc.CurrentID)
		} else {
			pass++
		}

	}

	println("Realtime Decrypt", pass, "out of", test, "tests passed.")

}
