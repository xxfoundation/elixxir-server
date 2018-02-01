package precomputation

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
	"testing"
)

func TestEncrypt(t *testing.T) {
	// NOTE: Does not test correctness

	test := 3
	pass := 0

	bs := uint64(3)

	round := globals.NewRound(bs)

	var im []services.Slot

	rng := cyclic.NewRandom(cyclic.NewInt(0), cyclic.NewInt(1000))

	grp := cyclic.NewGroup(cyclic.NewInt(101), cyclic.NewInt(23), cyclic.NewInt(55), rng)

	globals.Grp = &grp

	im = append(im, &SlotEncrypt{
		Slot:                     uint64(0),
		EncryptedMessageKeys:     cyclic.NewInt(int64(91)),
		PartialMessageCypherText: cyclic.NewInt(int64(73))})

	im = append(im, &SlotEncrypt{
		Slot:                     uint64(1),
		EncryptedMessageKeys:     cyclic.NewInt(int64(86)),
		PartialMessageCypherText: cyclic.NewInt(int64(87))})

	im = append(im, &SlotEncrypt{
		Slot:                     uint64(2),
		EncryptedMessageKeys:     cyclic.NewInt(int64(39)),
		PartialMessageCypherText: cyclic.NewInt(int64(50))})

	round.CypherPublicKey = cyclic.NewInt(30)

	round.Y_T[0] = cyclic.NewInt(53)
	round.Y_T[1] = cyclic.NewInt(24)
	round.Y_T[2] = cyclic.NewInt(61)

	round.T_INV[0] = cyclic.NewInt(52)
	round.T_INV[1] = cyclic.NewInt(68)
	round.T_INV[2] = cyclic.NewInt(11)

	expected := [][]*cyclic.Int{
		{cyclic.NewInt(16), cyclic.NewInt(86)},
		{cyclic.NewInt(90), cyclic.NewInt(88)},
		{cyclic.NewInt(32), cyclic.NewInt(66)},
	}

	dc := services.DispatchCryptop(&grp, Encrypt{}, nil, nil, round)

	for i := uint64(0); i < bs; i++ {
		dc.InChannel <- &(im[i])
		actual := <-dc.OutChannel

		act := (*actual).(*SlotEncrypt)

		expectedVal := expected[i]

		if expectedVal[0].Cmp(act.EncryptedMessageKeys) != 0 {
			t.Errorf("Test of Precomputation Encrypt's cryptop failed Keys Test on index: %v; "+
				"Expected: %v, Actual: %v", i, expectedVal[0].Text(10),
				act.EncryptedMessageKeys.Text(10))
		} else if expectedVal[1].Cmp(act.PartialMessageCypherText) != 0 {
			t.Errorf("Test of Precomputation Encrypt's cryptop failed Cypher Text Test on index: %v; "+
				"Expected: %v, Actual: %v", i, expectedVal[1].Text(10),
				act.PartialMessageCypherText.Text(10))
		} else {
			pass++
		}

	}

	println("Precomputation Encrypt", pass, "out of", test, "tests passed.")

}
