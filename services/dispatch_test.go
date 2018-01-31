package services

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/node"
	"testing"
)

type Test struct{}

type SlotTest struct {
	slot uint64

	A *cyclic.Int
}

func (ts SlotTest) SlotID() uint64 {
	return ts.slot
}

type KeysTest struct {
	R *cyclic.Int
}

func (cry Test) Run(g *cyclic.Group, in, out *SlotTest, keys *KeysTest) Slot {

	out.A.Add(in.A, keys.R)

	keys.R.Set(cyclic.NewInt(15))

	return out
}

func (cry Test) Build(g *cyclic.Group, face interface{}) *DispatchBuilder {

	round := face.(*node.Round)

	om := make([]Slot, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = &SlotTest{slot: i, A: cyclic.NewMaxInt()}
	}

	keys := make([]NodeKeys, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		keys[i] = &KeysTest{R: round.R[i]}
	}

	db := DispatchBuilder{BatchSize: round.BatchSize, Keys: &keys, Output: &om, G: g}

	return &db
}

func TestDispatchCryptop(t *testing.T) {

	test := 10
	pass := 0

	bs := uint64(4)

	round := node.NewRound(bs)

	var im []Slot

	i := uint64(0)
	for i < bs {
		im = append(im, &SlotTest{slot: uint64(i), A: cyclic.NewInt(int64(i + 1))})
		round.R[i] = cyclic.NewInt(int64(2 * (i + 1)))
		i++
	}

	result := []*cyclic.Int{
		cyclic.NewInt(3), cyclic.NewInt(6), cyclic.NewInt(9), cyclic.NewInt(12),
	}

	rng := cyclic.NewRandom(cyclic.NewInt(0), cyclic.NewInt(1000))

	grp := cyclic.NewGroup(cyclic.NewInt(11), cyclic.NewInt(5), cyclic.NewInt(12), rng)

	dc := DispatchCryptop(&grp, Test{}, nil, nil, round)

	if dc.IsAlive() {
		pass++
	} else {
		t.Errorf("IsAlive: Expected dispatch to be alive after initialization!")
	}

	for i := uint64(0); i < bs; i++ {
		dc.InChannel <- &im[i]
		trn := <-dc.OutChannel

		rtn := (*trn).(*SlotTest)

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

func TestDispatchController_IsAlive(t *testing.T) {

	round := node.NewRound(uint64(4))

	rng := cyclic.NewRandom(cyclic.NewInt(0), cyclic.NewInt(1000))

	grp := cyclic.NewGroup(cyclic.NewInt(11), cyclic.NewInt(5), cyclic.NewInt(12), rng)

	dc := DispatchCryptop(&grp, Test{}, nil, nil, round)

	if !dc.IsAlive() {
		t.Errorf("IsAlive: Expected dispatch to be alive after initialization!")
	}

	// Block until the dispatcher is dead
	// To not block until the dispatcher is dead, pass false to dc.Kill.
	dc.Kill(true)

	if dc.IsAlive() {
		t.Errorf("IsAlive: Expected dispatch to be dead after Kill signal!")
	}

}
