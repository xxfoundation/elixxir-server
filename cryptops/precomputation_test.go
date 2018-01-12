package cryptops

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/server"
	"gitlab.com/privategrity/server/services"
	"testing"
)

func TestPrecompGeneration(t *testing.T) {

	test := 102
	pass := 0

	bs := uint64(100)

	round := server.NewRound(bs)

	defaultInt := cyclic.NewInt(0)
	defaultInt.SetBytes(cyclic.Max4kBitInt)

	var im []*services.Message

	for i := uint64(0); i < bs; i++ {
		im = append(im, &services.Message{uint64(i), []*cyclic.Int{cyclic.NewInt(int64(0))}})
	}

	gen := cyclic.NewGen(cyclic.NewInt(0), cyclic.NewInt(1000))

	g := cyclic.NewGroup(cyclic.NewInt(11), cyclic.NewInt(5), gen)

	dc := services.DispatchCryptop(&g, PrecompGeneration{}, nil, nil, round)

	roundcnt := 0

	for i := uint64(0); i < bs; i++ {

		dc.InChannel <- im[i]
		rtn := <-dc.OutChannel

		if !validRound(round, defaultInt, i) {
			t.Errorf("Test of PrecompGeneration's random generation failed at index: %v ", i)
		} else if rtn.Slot != i {
			t.Errorf("Test of PrecompGeneration's output index failed at index: %v", i)
		} else {
			pass++
		}

		if round.Permutations[i] == i {
			roundcnt++
		}

	}

	if round.Z.Cmp(defaultInt) == 0 {
		t.Errorf("Test of PrecompGeneration's random generation of the Global Cypher Key failed")
	} else {
		pass++
	}

	if roundcnt > 20 {
		t.Errorf("Test of PrecompGeneration's shuffle failed")
	} else {
		pass++
	}

	println("PrecompGeneration", pass, "out of", test, "tests passed.")

}

func validRound(round *server.Round, cmped *cyclic.Int, i uint64) bool {
	if round.R[i].Cmp(cmped) == 0 {
		return false
	} else if round.S[i].Cmp(cmped) == 0 {
		return false
	} else if round.T[i].Cmp(cmped) == 0 {
		return false
	} else if round.U[i].Cmp(cmped) == 0 {
		return false
	} else if round.V[i].Cmp(cmped) == 0 {
		return false
	} else if round.R_INV[i].Cmp(cmped) == 0 {
		return false
	} else if round.S_INV[i].Cmp(cmped) == 0 {
		return false
	} else if round.T_INV[i].Cmp(cmped) == 0 {
		return false
	} else if round.U_INV[i].Cmp(cmped) == 0 {
		return false
	} else if round.V_INV[i].Cmp(cmped) == 0 {
		return false
	} else if round.Y_R[i].Cmp(cmped) == 0 {
		return false
	} else if round.Y_S[i].Cmp(cmped) == 0 {
		return false
	} else if round.Y_T[i].Cmp(cmped) == 0 {
		return false
	} else if round.Y_U[i].Cmp(cmped) == 0 {
		return false
	} else if round.Y_V[i].Cmp(cmped) == 0 {
		return false
	} else {
		return true
	}
}

func TestPrecompEncrypt(t *testing.T) {
	// NOTE: Does not test correctness

	test := 3
	pass := 0

	bs := uint64(3)

	round := server.NewRound(bs)

	var im []*services.Message

	gen := cyclic.NewGen(cyclic.NewInt(0), cyclic.NewInt(1000))

	g := cyclic.NewGroup(cyclic.NewInt(101), cyclic.NewInt(23), gen)

	im = append(im, &services.Message{uint64(0), []*cyclic.Int{
		cyclic.NewInt(int64(39)), cyclic.NewInt(int64(13)),
		cyclic.NewInt(int64(41)), cyclic.NewInt(int64(74)),
	}})

	im = append(im, &services.Message{uint64(1), []*cyclic.Int{
		cyclic.NewInt(int64(86)), cyclic.NewInt(int64(87)),
		cyclic.NewInt(int64(8)), cyclic.NewInt(int64(49)),
	}})

	im = append(im, &services.Message{uint64(2), []*cyclic.Int{
		cyclic.NewInt(int64(39)), cyclic.NewInt(int64(51)),
		cyclic.NewInt(int64(91)), cyclic.NewInt(int64(73)),
	}})

	server.G = cyclic.NewInt(55)
	round.G = cyclic.NewInt(30)

	round.S_INV[0] = cyclic.NewInt(53)
	round.S_INV[1] = cyclic.NewInt(24)
	round.S_INV[2] = cyclic.NewInt(61)

	round.V_INV[0] = cyclic.NewInt(52)
	round.V_INV[1] = cyclic.NewInt(68)
	round.V_INV[2] = cyclic.NewInt(11)

	round.Y_S[0] = cyclic.NewInt(98)
	round.Y_S[1] = cyclic.NewInt(7)
	round.Y_S[2] = cyclic.NewInt(32)

	round.Y_V[0] = cyclic.NewInt(23)
	round.Y_V[1] = cyclic.NewInt(16)
	round.Y_V[2] = cyclic.NewInt(17)

	expected := [][]*cyclic.Int{
		{cyclic.NewInt(44), cyclic.NewInt(19), cyclic.NewInt(17)},
		{cyclic.NewInt(53), cyclic.NewInt(65), cyclic.NewInt(17)},
		{cyclic.NewInt(44), cyclic.NewInt(59), cyclic.NewInt(17)},
	}

	dc := services.DispatchCryptop(&g, PrecompEncrypt{}, nil, nil, round)

	for i := uint64(0); i < bs; i++ {
		dc.InChannel <- im[i]
		rtn := <-dc.OutChannel

		actual := expected[i]

		valid := true

		for j := 0; j < 3; j++ {
			valid = valid && (actual[j].Cmp(rtn.Data[j]) == 0)
		}

		if !valid {
			t.Errorf("Test of PrecompEncrypt's cryptop failed on index: %v", i)
		} else {
			pass++
		}

	}

	println("PrecompEncrypt", pass, "out of", test, "tests passed.")

}
