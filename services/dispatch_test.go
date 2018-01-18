package services

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/node"
	"testing"
)

type testMessage struct {
	slot uint64

	A *cyclic.Int
}

func (tm testMessage) Slot() uint64 {
	return tm.slot
}

type testKeys struct {
	R *cyclic.Int
}

type testCryptop struct{}

func (cry testCryptop) Run(g *cyclic.Group, in, out *testMessage, keys *testKeys) Message {

	out.A.Add(in.A, keys.R)

	keys.R.Set(cyclic.NewInt(15))

	return out
}

func (cry testCryptop) Build(g *cyclic.Group, face interface{}) *DispatchBuilder {

	round := face.(*node.Round)

	om := make([]Message, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = &testMessage{slot: i, A: cyclic.NewMaxInt()}
	}

	keys := make([]NodeKeys, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		keys[i] = &testKeys{R: round.R[i]}
	}

	db := DispatchBuilder{BatchSize: round.BatchSize, Keys: &keys, Output: &om, G: g}

	return &db
}

func TestDispatchCryptop(t *testing.T) {

	test := 10
	pass := 0

	bs := uint64(4)

	round := node.NewRound(bs)

	var im []Message

	i := uint64(0)
	for i < bs {
		im = append(im, &testMessage{slot: uint64(i), A: cyclic.NewInt(int64(i + 1))})
		round.R[i] = cyclic.NewInt(int64(2 * (i + 1)))
		i++
	}

	result := []*cyclic.Int{
		cyclic.NewInt(3), cyclic.NewInt(6), cyclic.NewInt(9), cyclic.NewInt(12),
	}

	rng := cyclic.NewRandom(cyclic.NewInt(0), cyclic.NewInt(1000))

	grp := cyclic.NewGroup(cyclic.NewInt(11), cyclic.NewInt(5), cyclic.NewInt(12), rng)

	dc := DispatchCryptop(&grp, testCryptop{}, nil, nil, round)

	if dc.IsAlive() {
		pass++
	} else {
		t.Errorf("IsAlive: Expected dispatch to be alive after initialization!")
	}

	for i := uint64(0); i < bs; i++ {
		dc.InChannel <- &im[i]
		trn := <-dc.OutChannel

		rtn := (*trn).(*testMessage)

		if rtn.A.Cmp(result[i]) != 0 {
			t.Errorf("Test of Dispatcher failed at index: %v Expected: %v;",
				" Actual: %v", i, result[i].Text(10), rtn.A.Text(10))
		} else {
			pass++
		}

		if round.R[i].Int64() != 15 {
			t.Errorf("Test of Dispatcher pass by reference failed at index: %v Expected: %v;",
				" Actual: %v", i, 15, round.R[i].Text(10))
		} else {
			pass++
		}

	}

	if !dc.IsAlive() {
		pass++
	} else {
		t.Errorf("IsAlive: Expected dispatch to be dead after channels closed!")
	}

	println("Dispatcher", pass, "out of", test, "tests passed.")

}
