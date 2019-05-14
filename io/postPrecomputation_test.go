package io

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/server/server/round"
	"testing"
)

func TestPostPrecompResult_Errors(t *testing.T) {
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2), large.NewInt(97))
	r := round.NewBuffer(grp, 5, 5)

	// If the number of slots doesn't match the batch, there should be an error
	err := PostPrecompResult(r, grp, []*mixmessages.Slot{})
	if err == nil {
		t.Error("No error from batch size mismatch")
	}
}

func TestPostPrecompResult(t *testing.T) {
	// This test actually overwrites the precomputations for a round
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2), large.NewInt(97))
	const bs = 5
	r := round.NewBuffer(grp, bs, bs)

	// There should be no error in this case, because there are enough slots
    var slots []*mixmessages.Slot
	const start = 2
	for precompValue := start; precompValue < bs + start; precompValue++ {
        slots = append(slots, &mixmessages.Slot{
			PartialMessageCypherText:        grp.NewInt(int64(precompValue)).
				Bytes(),
			PartialAssociatedDataCypherText: grp.NewInt(int64(precompValue+bs)).
				Bytes(),
		})
	}

	err := PostPrecompResult(r, grp, slots)
	if err != nil {
		t.Error(err)
	}

	// Then, the slots in the round buffer should be set to those integers
	for precompValue := start; precompValue < bs + start; precompValue++ {
		index := uint32(precompValue- start)
		messagePrecomp := r.MessagePrecomputation.Get(index)
		if messagePrecomp.Cmp(grp.NewInt(int64(precompValue))) != 0 {
			t.Errorf("Message precomp didn't match at index %v", index)
		}
		adPrecomp := r.ADPrecomputation.Get(index)
		if adPrecomp.Cmp(grp.NewInt(int64(precompValue+bs))) != 0 {
			t.Errorf("Associated data precomp didn't match at index %v", index)
		}
	}
}
