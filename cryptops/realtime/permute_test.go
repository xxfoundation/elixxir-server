package realtime

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/server"
	"gitlab.com/privategrity/server/services"
	"testing"
)

func TestPrecompPermutation(t *testing.T) {

	test := 9
	pass := 0

	bs := uint64(3)

	round := server.NewRound(bs)

	var im []*services.Message

	gen := cyclic.NewGen(cyclic.NewInt(0), cyclic.NewInt(1000))

	g := cyclic.NewGroup(cyclic.NewInt(101), cyclic.NewInt(23), gen)

	im = append(im, &services.Message{uint64(0), []*cyclic.Int{
		cyclic.NewInt(int64(39)), cyclic.NewInt(int64(13)),
	}})

	im = append(im, &services.Message{uint64(1), []*cyclic.Int{
		cyclic.NewInt(int64(86)), cyclic.NewInt(int64(87)),
	}})

	im = append(im, &services.Message{uint64(2), []*cyclic.Int{
		cyclic.NewInt(int64(39)), cyclic.NewInt(int64(51)),
		cyclic.NewInt(int64(91)), cyclic.NewInt(int64(73)),
	}})

	server.G = cyclic.NewInt(55)
	round.CypherPublicKey = cyclic.NewInt(30)

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

	dc := services.DispatchCryptop(&g, RealPermute{}, nil, nil, round)

	for i := uint64(0); i < bs; i++ {
		dc.InChannel <- im[i]
		rtn := <-dc.OutChannel

		result := results[i]

		for j := 0; j < 2; j++ {
			if result[j].Cmp(rtn.Data[j]) != 0 {
				t.Errorf("Test of RealPermute's cryptop failed on index: %v on value: %v.  Expected: %v Received: %v ",
					i, j, result[j].Text(10), rtn.Data[j].Text(10))
			} else {
				pass++
			}
		}

		if rtn.Slot == i {
			t.Errorf("Test of RealPermute's permute failed on index: %v", i)
		} else {
			pass++
		}

	}

	println("RealPermute", pass, "out of", test, "tests passed.")

}
