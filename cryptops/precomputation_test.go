package cryptops

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/server"
	"gitlab.com/privategrity/server/services"
	"testing"
)

func TestPrecompGeneration(t *testing.T) {

	test := 4
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

	for i := uint64(0); i < bs; i++ {

		dc.InChannel <- im[i]
		rtn := <-dc.OutChannel

		rtn.Slot = uint64(0)

		if !validRound(round, defaultInt, i) {
			t.Errorf("Test of PrecompGeneration failed at index: %v ", i)
		} else {
			pass++
		}

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
