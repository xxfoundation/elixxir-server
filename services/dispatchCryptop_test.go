////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package services

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"testing"
)

type Test struct{}

type SlotTest struct {
	slot uint64

	A *large.Int
}

func (ts SlotTest) SlotID() uint64 {
	return ts.slot
}

type KeysTest struct {
	R *cyclic.Int
}

func (cry Test) Run(grp *cyclic.Group, in, out *SlotTest, keys *KeysTest) Slot {

	out.A.Add(in.A, keys.R.GetLargeInt())

	grp.Set(keys.R, grp.NewInt(15))

	return out
}

func (cry Test) Build(grp *cyclic.Group, face interface{}) *DispatchBuilder {

	bs := uint64(4)

	round := face.([]*cyclic.Int)

	om := make([]Slot, bs)

	for i := uint64(0); i < bs; i++ {
		om[i] = &SlotTest{slot: i, A: large.NewMaxInt()}
	}

	keys := make([]NodeKeys, bs)

	for i := uint64(0); i < bs; i++ {
		keys[i] = &KeysTest{R: round[i]}
	}

	db := DispatchBuilder{BatchSize: bs, Keys: &keys, Output: &om, G: grp}

	return &db
}

func TestDispatchCryptop(t *testing.T) {

	test := 10
	pass := 0

	grp := cyclic.NewGroup(large.NewInt(11), large.NewInt(5), large.NewInt(12))

	bs := uint64(4)

	round := make([]*cyclic.Int, bs)

	var im []Slot

	i := uint64(0)
	for i < bs {
		im = append(im, &SlotTest{slot: uint64(i), A: large.NewInt(int64(i + 1))})
		round[i] = grp.NewInt(int64(2 * (i + 1)))
		i++
	}

	result := []*large.Int{
		large.NewInt(18), large.NewInt(21), large.NewInt(24), large.NewInt(27),
	}

	dc1 := DispatchCryptop(&grp, Test{}, nil, nil, round)
	dc2 := DispatchCryptop(&grp, Test{}, dc1.OutChannel, nil, round)

	if dc1.IsAlive() && dc2.IsAlive() {
		pass++
	} else {
		t.Errorf("IsAlive: Expected dispatch to be alive after initialization!")
	}

	for i := uint64(0); i < bs; i++ {
		dc1.InChannel <- &im[i]
		trn := <-dc2.OutChannel

		rtn := (*trn).(*SlotTest)

		if rtn.A.Cmp(result[i]) != 0 {
			t.Errorf("Test of Dispatcher failed at index: %v Expected: %v;"+
				" Actual: %v", i, result[i].Text(10), rtn.A.Text(10))
		} else {
			pass++
		}

		if round[i].GetLargeInt().Int64() != 15 {
			t.Errorf("Test of Dispatcher pass by reference failed at index"+
				": %v Expected: %v;"+
				" Actual: %v", i, 15, round[i].Text(10))
		} else {
			pass++
		}

	}

	if !dc1.IsAlive() && !dc2.IsAlive() {
		pass++
	} else {
		t.Errorf("IsAlive: Expected dispatch to be dead after channels closed!")
	}

	println("Dispatcher", pass, "out of", test, "tests passed.")

}

func TestDispatchController_IsAlive(t *testing.T) {

	round := make([]*cyclic.Int, 4)

	grp := cyclic.NewGroup(large.NewInt(11), large.NewInt(5), large.NewInt(12))

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
