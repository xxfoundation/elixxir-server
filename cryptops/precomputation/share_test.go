package precomputation

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/server"
	"gitlab.com/privategrity/server/services"
	"testing"
)

func TestPrecompShare(t *testing.T) {
	// NOTE: Does not test correctness

	test := 3
	pass := 0

	bs := uint64(3)

	round := server.NewRound(bs)

	var im []*services.Message

	gen := cyclic.NewGen(cyclic.NewInt(0), cyclic.NewInt(1000))

	g := cyclic.NewGroup(cyclic.NewInt(101), cyclic.NewInt(23), gen)

	im = append(im, &services.Message{uint64(0), []*cyclic.Int{
		cyclic.NewInt(int64(39))}})

	im = append(im, &services.Message{uint64(1), []*cyclic.Int{
		cyclic.NewInt(int64(86))}})

	im = append(im, &services.Message{uint64(2), []*cyclic.Int{
		cyclic.NewInt(int64(66))}})

	round.Z = cyclic.NewInt(53)

	expected := [][]*cyclic.Int{
		{cyclic.NewInt(69)},
		{cyclic.NewInt(42)},
		{cyclic.NewInt(51)},
	}

	dc := services.DispatchCryptop(&g, PrecompShare{}, nil, nil, round)

	for i := uint64(0); i < bs; i++ {
		dc.InChannel <- im[i]
		rtn := <-dc.OutChannel

		result := expected[i]

		valid := true

		for j := 0; j < 1; j++ {
			if result[j].Cmp(rtn.Data[j]) != 0 {
				t.Errorf("Test of PrecompShare's cryptop failed on index: %v on value: %v.  Expected: %v Received: %v ",
					i, j, result[j].Text(10), rtn.Data[j].Text(10))
			} else {
				pass++
			}
		}

		if !valid {
			t.Errorf("Test of PrecompShare's cryptop failed on index: %v", i)
		} else {
			pass++
		}

	}

	println("PrecompShare", pass, "out of", test, "tests passed.")

}
