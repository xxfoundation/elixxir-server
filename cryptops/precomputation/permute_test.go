package cryptops

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/server"
	"gitlab.com/privategrity/server/services"
	"testing"
)

func TestPrecompPermutation(t *testing.T) {

	test := 15
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

	round.Permutations[0] = 1
	round.Permutations[1] = 2
	round.Permutations[2] = 0

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

	results := [][]*cyclic.Int{
		{cyclic.NewInt(31), cyclic.NewInt(62), cyclic.NewInt(74), cyclic.NewInt(98)},
		{cyclic.NewInt(96), cyclic.NewInt(31), cyclic.NewInt(73), cyclic.NewInt(77)},
		{cyclic.NewInt(19), cyclic.NewInt(72), cyclic.NewInt(66), cyclic.NewInt(94)},
	}

	dc := services.DispatchCryptop(&g, PrecompPermute{}, nil, nil, round)

	for i := uint64(0); i < bs; i++ {
		dc.InChannel <- im[i]
		rtn := <-dc.OutChannel

		result := results[i]

		for j := 0; j < 4; j++ {
			if result[j].Cmp(rtn.Data[j]) != 0 {
				t.Errorf("Test of PrecompPermutation's cryptop failed on index: %v on value: %v.  Expected: %v Received: %v ",
					i, j, result[j].Text(10), rtn.Data[j].Text(10))
			} else {
				pass++
			}
		}

		if rtn.Slot == i {
			t.Errorf("Test of PrecompPermutation's permute failed on index: %v", i)
		} else {
			pass++
		}

	}

	println("PrecompPermutation", pass, "out of", test, "tests passed.")

}
