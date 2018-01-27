package realtime

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/node"
	"gitlab.com/privategrity/server/services"
	"testing"
)

func TestRealTimePermute(t *testing.T) {

	test := 6
	pass := 0

	bs := uint64(3)

	round := node.NewRound(bs)

	var im []services.Slot

	rng := cyclic.NewRandom(cyclic.NewInt(0), cyclic.NewInt(1000))

	grp := cyclic.NewGroup(cyclic.NewInt(101), cyclic.NewInt(23),
		cyclic.NewInt(29), rng)

	im = append(im, &SlotPermute{
		Slot:                 uint64(0),
		EncryptedMessage:     cyclic.NewInt(int64(39)),
		EncryptedRecipientID: cyclic.NewInt(int64(13))})

	im = append(im, &SlotPermute{
		Slot:                 uint64(1),
		EncryptedMessage:     cyclic.NewInt(int64(86)),
		EncryptedRecipientID: cyclic.NewInt(int64(87))})

	im = append(im, &SlotPermute{
		Slot:                 uint64(2),
		EncryptedMessage:     cyclic.NewInt(int64(39)),
		EncryptedRecipientID: cyclic.NewInt(int64(51))})

	round.Permutations[0] = 1
	round.Permutations[1] = 2
	round.Permutations[2] = 0

	round.S[0] = cyclic.NewInt(53)
	round.S[1] = cyclic.NewInt(24)
	round.S[2] = cyclic.NewInt(61)

	round.V[0] = cyclic.NewInt(52)
	round.V[1] = cyclic.NewInt(68)
	round.V[2] = cyclic.NewInt(11)

	results := [][]*cyclic.Int{
		{cyclic.NewInt(47), cyclic.NewInt(70)},
		{cyclic.NewInt(44), cyclic.NewInt(58)},
		{cyclic.NewInt(56), cyclic.NewInt(56)},
	}

	dc := services.DispatchCryptop(&grp, Permute{}, nil, nil, round)

	for i := uint64(0); i < bs; i++ {
		dc.InChannel <- &im[i]
		trn := <-dc.OutChannel

		rtn := (*trn).(*SlotPermute)
		result := results[i]

		if result[0].Cmp(rtn.EncryptedMessage) != 0 ||
			result[1].Cmp(rtn.EncryptedRecipientID) != 0 {
			t.Errorf("Test of RealPermute's cryptop failed on index: %v"+
				" Expected: %v,%v Received: %v,%v ", i,
				result[0].Text(10), result[1].Text(10),
				rtn.EncryptedMessage.Text(10), rtn.EncryptedRecipientID.Text(10))
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

	var im []*SlotPermute
	var om []*SlotPermute

	rng := cyclic.NewRandom(cyclic.NewInt(0), cyclic.NewInt(1000))

	grp := cyclic.NewGroup(cyclic.NewInt(101), cyclic.NewInt(23),
		cyclic.NewInt(29), rng)

	im = append(im, &SlotPermute{
		Slot:                 uint64(0),
		EncryptedMessage:     cyclic.NewInt(int64(39)),
		EncryptedRecipientID: cyclic.NewInt(int64(13))})

	im = append(im, &SlotPermute{
		Slot:                 uint64(1),
		EncryptedMessage:     cyclic.NewInt(int64(86)),
		EncryptedRecipientID: cyclic.NewInt(int64(87))})

	im = append(im, &SlotPermute{
		Slot:                 uint64(2),
		EncryptedMessage:     cyclic.NewInt(int64(39)),
		EncryptedRecipientID: cyclic.NewInt(int64(51))})

	om = append(om, &SlotPermute{
		Slot:                 uint64(1),
		EncryptedMessage:     cyclic.NewInt(int64(0)),
		EncryptedRecipientID: cyclic.NewInt(int64(0))})

	om = append(om, &SlotPermute{
		Slot:                 uint64(2),
		EncryptedMessage:     cyclic.NewInt(int64(0)),
		EncryptedRecipientID: cyclic.NewInt(int64(0))})

	om = append(om, &SlotPermute{
		Slot:                 uint64(0),
		EncryptedMessage:     cyclic.NewInt(int64(0)),
		EncryptedRecipientID: cyclic.NewInt(int64(0))})

	keys := []KeysPermute{
		{
			S: cyclic.NewInt(53),
			V: cyclic.NewInt(52)},
		{
			S: cyclic.NewInt(24),
			V: cyclic.NewInt(68)},
		{
			S: cyclic.NewInt(61),
			V: cyclic.NewInt(11)},
	}

	results := [][]*cyclic.Int{
		{cyclic.NewInt(47), cyclic.NewInt(70)},
		{cyclic.NewInt(44), cyclic.NewInt(58)},
		{cyclic.NewInt(56), cyclic.NewInt(56)},
	}

	permute := Permute{}

	for i := uint64(0); i < bs; i++ {
		permute.Run(&grp, im[i], om[i], &keys[i])
	}

	for i := uint64(0); i < bs; i++ {
		if results[i][0].Cmp(om[i].EncryptedMessage) != 0 ||
			results[i][1].Cmp(om[i].EncryptedRecipientID) != 0 {
			t.Errorf("%v - Expected: %v,%v Got: %v,%v",
				results[i][0].Text(10), results[i][1].Text(10),
				om[i].EncryptedMessage.Text(10),
				om[i].EncryptedRecipientID.Text(10))
		}
	}
}
