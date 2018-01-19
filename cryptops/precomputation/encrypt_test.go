package precomputation

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/node"
	"gitlab.com/privategrity/server/services"
	"testing"
)

func TestEncrypt(t *testing.T) {
	// NOTE: Does not test correctness

	test := 3
	pass := 0

	bs := uint64(3)

	round := node.NewRound(bs)

	rng := cyclic.NewRandom(cyclic.NewInt(0), cyclic.NewInt(1000))

	grp := cyclic.NewGroup(cyclic.NewInt(101), cyclic.NewInt(23), cyclic.NewInt(55), rng)

	node.Grp = &grp

	var im []services.Slot

	im = append(im, &SlotEncrypt{slot: 0, EncryptedMessageKeys: cyclic.NewInt(55),
		PartialMessageCypherText: cyclic.NewInt(64)})

	im = append(im, &services.Message{uint64(1), []*cyclic.Int{
		cyclic.NewInt(int64(86)), cyclic.NewInt(int64(87)),
		cyclic.NewInt(int64(8)), cyclic.NewInt(int64(49)),
	}})

	im = append(im, &services.Message{uint64(2), []*cyclic.Int{
		cyclic.NewInt(int64(39)), cyclic.NewInt(int64(51)),
		cyclic.NewInt(int64(91)), cyclic.NewInt(int64(73)),
	}})

	round.CypherPublicKey = cyclic.NewInt(30)

	round.Y_T[0] = cyclic.NewInt(53)
	round.Y_T[1] = cyclic.NewInt(24)
	round.Y_T[2] = cyclic.NewInt(61)

	round.T_INV[0] = cyclic.NewInt(52)
	round.T_INV[1] = cyclic.NewInt(68)
	round.T_INV[2] = cyclic.NewInt(11)

	expected := [][]*cyclic.Int{
		{cyclic.NewInt(79), cyclic.NewInt(25)},
		{cyclic.NewInt(90), cyclic.NewInt(88)},
		{cyclic.NewInt(32), cyclic.NewInt(35)},
	}

	dc := services.DispatchCryptop(&grp, PrecompEncrypt{}, nil, nil, round)

	for i := uint64(0); i < bs; i++ {
		dc.InChannel <- im[i]
		actual := <-dc.OutChannel

		expectedVal := expected[i]

		valid := true

		for j := 0; j < 2; j++ {
			valid = valid && (expectedVal[j].Cmp(actual.Data[j]) == 0)
		}

		if !valid {
			t.Errorf("Test of PrecompEncrypt's cryptop failed on index: %v", i)
		} else {
			pass++
		}

	}

	println("PrecompEncrypt", pass, "out of", test, "tests passed.")

}
