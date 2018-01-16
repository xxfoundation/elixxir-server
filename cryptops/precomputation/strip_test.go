package precomputation

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/server"
	"gitlab.com/privategrity/server/services"
	"testing"
)

func TestPrecompStrip(t *testing.T) {
	// NOTE: Does not test correctness

	test := 3
	pass := 0

	bs := uint64(3)

	round := server.NewRound(bs)

	var im []*services.Message

	rng := cyclic.NewRandom(cyclic.NewInt(0), cyclic.NewInt(1000))

	grp := cyclic.NewGroup(cyclic.NewInt(101), cyclic.NewInt(23), cyclic.NewInt(29), rng)

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

	expected := [][]*cyclic.Int{
		{cyclic.NewInt(34), cyclic.NewInt(56)},
		{cyclic.NewInt(75), cyclic.NewInt(44)},
		{cyclic.NewInt(79), cyclic.NewInt(23)},
	}

	dc := services.DispatchCryptop(&grp, PrecompStrip{}, nil, nil, round)

	for i := uint64(0); i < bs; i++ {
		dc.InChannel <- im[i]
		actual := <-dc.OutChannel

		expectedVal := expected[i]

		valid := true

		for j := 0; j < 2; j++ {
			valid = valid && (expectedVal[j].Cmp(actual.Data[j]) == 0)
		}

		if !valid {
			t.Errorf("Test of PrecompStrip's cryptop failed on index: %v", i)
		} else {
			pass++
		}

	}

	println("PrecompStrip", pass, "out of", test, "tests passed.")

}
