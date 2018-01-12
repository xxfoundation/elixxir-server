package precomputation

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
