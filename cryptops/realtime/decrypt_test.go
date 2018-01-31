package realtime

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/node"
	"gitlab.com/privategrity/server/services"
	"testing"
)

func TestDecrypt(t *testing.T) {
	// NOTE: Does not test correctness

	test := 9
	pass := 0

	bs := uint64(3)

	round := node.NewRound(bs)

	rng := cyclic.NewRandom(cyclic.NewInt(0), cyclic.NewInt(1000))

	grp := cyclic.NewGroup(cyclic.NewInt(101), cyclic.NewInt(23), cyclic.NewInt(27), rng)

	senderIds := [3]uint64{uint64(5), uint64(7), uint64(9)}

	var im []services.Slot

	im = append(im, &SlotDecryptIn{
		Slot:                 uint64(0),
		SenderID:             senderIds[0],
		EncryptedMessage:     cyclic.NewInt(int64(39)),
		TransmissionKey:      cyclic.NewInt(int64(65)),
		EncryptedRecipientID: cyclic.NewInt(7)})

	im = append(im, &SlotDecryptIn{
		Slot:                 uint64(1),
		SenderID:             senderIds[1],
		EncryptedMessage:     cyclic.NewInt(int64(86)),
		TransmissionKey:      cyclic.NewInt(int64(44)),
		EncryptedRecipientID: cyclic.NewInt(51)})

	im = append(im, &SlotDecryptIn{
		Slot:                 uint64(2),
		SenderID:             senderIds[2],
		EncryptedMessage:     cyclic.NewInt(int64(66)),
		TransmissionKey:      cyclic.NewInt(int64(94)),
		EncryptedRecipientID: cyclic.NewInt(23)})

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

		rtnXtc := (*rtn).(*SlotDecryptOut)

		for j := 0; j < 1; j++ {
			// Test EncryptedMessage results
			if result[j].Cmp(rtnXtc.EncryptedMessage) != 0 {
				t.Errorf("Test of RealtimeDecrypt's EncryptedMessage output "+
					"failed on index: %v on value: %v.  Expected: %v Received: %v ",
					i, j, result[j].Text(10), rtnXtc.EncryptedMessage.Text(10))
			} else {
				pass++
			}
			// Test EncryptedRecipientID results
			if result[j+1].Cmp(rtnXtc.EncryptedRecipientID) != 0 {
				t.Errorf("Test of RealtimeDecrypt's EncryptedRecipientID output "+
					"failed on index: %v on value: %v.  Expected: %v Received: %v ",
					i, j+1, result[j+1].Text(10), rtnXtc.EncryptedRecipientID.Text(10))
			} else {
				pass++
			}
		}

		// Test SenderID pass through
		if senderIds[i] != rtnXtc.SenderID {
			t.Errorf("Test of RealtimeDecrypt's SenderID ouput failed on index %v.  Expected: %v Received: %v ",
				i, senderIds[i], rtnXtc.SenderID)
		} else {
			pass++
		}

	}

	println("Realtime Decrypt", pass, "out of", test, "tests passed.")

}
