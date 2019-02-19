////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package realtime

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/services"
	"testing"
)

func TestDecrypt(t *testing.T) {
	// NOTE: Does not test correctness

	test := 9
	pass := 0

	bs := uint64(3)

	round := globals.NewRound(bs)

	rng := cyclic.NewRandom(cyclic.NewInt(0), cyclic.NewInt(1000))

	grp := cyclic.NewGroup(cyclic.NewInt(101), cyclic.NewInt(23), cyclic.NewInt(27), rng)

	senderIds := [3]*id.User{id.NewUserFromUint(5, t),
		id.NewUserFromUint(7, t),
		id.NewUserFromUint(9, t),
	}

	var im []services.Slot

	im = append(im, &Slot{
		Slot:               uint64(0),
		CurrentID:          senderIds[0],
		Message:            cyclic.NewInt(int64(39)),
		CurrentKey:         cyclic.NewInt(int64(65)),
		EncryptedRecipient: cyclic.NewInt(7)})

	im = append(im, &Slot{
		Slot:               uint64(1),
		CurrentID:          senderIds[1],
		Message:            cyclic.NewInt(int64(86)),
		CurrentKey:         cyclic.NewInt(int64(44)),
		EncryptedRecipient: cyclic.NewInt(51)})

	im = append(im, &Slot{
		Slot:               uint64(2),
		CurrentID:          senderIds[2],
		Message:            cyclic.NewInt(int64(66)),
		CurrentKey:         cyclic.NewInt(int64(94)),
		EncryptedRecipient: cyclic.NewInt(23)})

	// Set the keys
	round.R[0] = cyclic.NewInt(52)
	round.R[1] = cyclic.NewInt(68)
	round.R[2] = cyclic.NewInt(11)

	round.U[0] = cyclic.NewInt(67)
	round.U[1] = cyclic.NewInt(88)
	round.U[2] = cyclic.NewInt(20)

	expected := [][]*cyclic.Int{
		{cyclic.NewInt(15), cyclic.NewInt(84)},
		{cyclic.NewInt(65), cyclic.NewInt(17)},
		{cyclic.NewInt(69), cyclic.NewInt(12)},
	}

	dc := services.DispatchCryptop(&grp, Decrypt{}, nil, nil, round)

	for i := uint64(0); i < bs; i++ {
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
			// Test EncryptedRecipientID results
			if result[j+1].Cmp(rtnXtc.EncryptedRecipient) != 0 {
				t.Errorf("Test of RealtimeDecrypt's EncryptedRecipientID output "+
					"failed on index: %v on value: %v.  Expected: %v Received: %v ",
					i, j+1, result[j+1].Text(10), rtnXtc.EncryptedRecipient.Text(10))
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
