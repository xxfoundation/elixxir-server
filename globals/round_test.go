////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"sync"
	"testing"
	"time"
)

// Grp is the global cyclic group used by cMix
var Group *cyclic.Group

// InitCrypto sets up the cryptographic constants for cMix
func InitCrypto() {

	base := 16

	pString := "9DB6FB5951B66BB6FE1E140F1D2CE5502374161FD6538DF1648218642F0B5C48" +
		"C8F7A41AADFA187324B87674FA1822B00F1ECF8136943D7C55757264E5A1A44F" +
		"FE012E9936E00C1D3E9310B01C7D179805D3058B2A9F4BB6F9716BFE6117C6B5" +
		"B3CC4D9BE341104AD4A80AD6C94E005F4B993E14F091EB51743BF33050C38DE2" +
		"35567E1B34C3D6A5C0CEAA1A0F368213C3D19843D0B4B09DCB9FC72D39C8DE41" +
		"F1BF14D4BB4563CA28371621CAD3324B6A2D392145BEBFAC748805236F5CA2FE" +
		"92B871CD8F9C36D3292B5509CA8CAA77A2ADFC7BFD77DDA6F71125A7456FEA15" +
		"3E433256A2261C6A06ED3693797E7995FAD5AABBCFBE3EDA2741E375404AE25B"

	gString := "5C7FF6B06F8F143FE8288433493E4769C4D988ACE5BE25A0E24809670716C613" +
		"D7B0CEE6932F8FAA7C44D2CB24523DA53FBE4F6EC3595892D1AA58C4328A06C4" +
		"6A15662E7EAA703A1DECF8BBB2D05DBE2EB956C142A338661D10461C0D135472" +
		"085057F3494309FFA73C611F78B32ADBB5740C361C9F35BE90997DB2014E2EF5" +
		"AA61782F52ABEB8BD6432C4DD097BC5423B285DAFB60DC364E8161F4A2A35ACA" +
		"3A10B1C4D203CC76A470A33AFDCBDD92959859ABD8B56E1725252D78EAC66E71" +
		"BA9AE3F1DD2487199874393CD4D832186800654760E1E34C09E4D155179F9EC0" +
		"DC4473F996BDCE6EED1CABED8B6F116F7AD9CF505DF0F998E34AB27514B0FFE7"

	qString := "F2C3119374CE76C9356990B465374A17F23F9ED35089BD969F61C6DDE9998C1F"
	p := large.NewIntFromString(pString, base)
	g := large.NewIntFromString(gString, base)
	q := large.NewIntFromString(qString, base)

	grpObject := cyclic.NewGroup(p, g, q)
	Group = grpObject
}

// TestNewRound tests that the round constructor really only returns
// an empty round with everything initialized to 0.
func TestNewRound(t *testing.T) {
	InitCrypto()

	var actual *Round
	size := uint64(42)
	actual = NewRound(size, Group)

	zero := Group.NewInt(1)
	zero = Group.NewMaxInt()

	if zero.Cmp(actual.CypherPublicKey) != 0 {
		t.Errorf("Test of NewRound() found Round CypherPublicKey is not set to Max4kBitInt")
	}
	if zero.Cmp(actual.Z) != 0 {
		t.Errorf("Test of NewRound() found Round Z is not set to Max4kBitInt")
	}

	if actual.BatchSize != size {
		t.Errorf("Test of NewRound() found Round BatchSize is not 42")
	}

	if actual.GetPhase() != OFF {
		t.Errorf("Test of NewRound() found Phase is %v instead of OFF", actual.GetPhase().String())
	}

	for i := uint64(0); i < size; i++ {
		if zero.Cmp(actual.R[i]) != 0 {
			t.Errorf("Test of NewRound() found Round R[%d] is not set to Max4kBitInt", i)
		}
		if zero.Cmp(actual.S[i]) != 0 {
			t.Errorf("Test of NewRound() found Round S[%d] is not set to Max4kBitInt", i)
		}
		if zero.Cmp(actual.T[i]) != 0 {
			t.Errorf("Test of NewRound() found Round T[%d] is not set to Max4kBitInt", i)
		}
		if zero.Cmp(actual.V[i]) != 0 {
			t.Errorf("Test of NewRound() found Round V[%d] is not set to Max4kBitInt", i)
		}
		if zero.Cmp(actual.U[i]) != 0 {
			t.Errorf("Test of NewRound() found Round U[%d] is not set to Max4kBitInt", i)
		}
		if zero.Cmp(actual.R_INV[i]) != 0 {
			t.Errorf("Test of NewRound() found Round R_INV[%d] is not set to Max4kBitInt", i)
		}
		if zero.Cmp(actual.S_INV[i]) != 0 {
			t.Errorf("Test of NewRound() found Round S_INV[%d] is not set to Max4kBitInt", i)
		}
		if zero.Cmp(actual.T_INV[i]) != 0 {
			t.Errorf("Test of NewRound() found Round T_INV[%d] is not set to Max4kBitInt", i)
		}
		if zero.Cmp(actual.V_INV[i]) != 0 {
			t.Errorf("Test of NewRound() found Round V_INV[%d] is not set to Max4kBitInt", i)
		}
		if zero.Cmp(actual.U_INV[i]) != 0 {
			t.Errorf("Test of NewRound() found Round U_INV[%d] is not set to Max4kBitInt", i)
		}
		if actual.Permutations[i] != i {
			t.Errorf("Test of NewRound() found Round Permutations[%d] is not set to its own permutation", i)
		}
		if zero.Cmp(actual.Y_R[i]) != 0 {
			t.Errorf("Test of NewRound() found Round Y_R[%d] is not set to Max4kBitInt", i)
		}
		if zero.Cmp(actual.Y_S[i]) != 0 {
			t.Errorf("Test of NewRound() found Round Y_S[%d] is not set to Max4kBitInt", i)
		}
		if zero.Cmp(actual.Y_T[i]) != 0 {
			t.Errorf("Test of NewRound() found Round Y_T[%d] is not set to Max4kBitInt", i)
		}
		if zero.Cmp(actual.Y_V[i]) != 0 {
			t.Errorf("Test of NewRound() found Round Y_V[%d] is not set to Max4kBitInt", i)
		}
		if zero.Cmp(actual.Y_U[i]) != 0 {
			t.Errorf("Test of NewRound() found Round Y_U[%d] is not set to Max4kBitInt", i)
		}
	}
}

func TestGetPhase(t *testing.T) {
	round := &Round{}
	round.phaseCond = &sync.Cond{L: &sync.Mutex{}}

	for p := OFF; p < NUM_PHASES; p++ {
		round.phase = p

		if round.GetPhase() != p {
			t.Errorf("Test of GetPhase() failed: Received: %v, Expected: %v", round.GetPhase().String(), p.String())
		}

	}

}

func TestNewRoundWithPhase(t *testing.T) {
	var actual *Round
	size := uint64(6)

	for p := OFF; p < NUM_PHASES; p++ {
		actual = NewRoundWithPhase(size, p, Group)

		zero := Group.NewInt(1)
		Group.Set(zero, Group.NewMaxInt())

		if zero.Cmp(actual.CypherPublicKey) != 0 {
			t.Errorf("Test of NewRoundWithPhase() found Round CypherPublicKey is not set to Max4kBitInt for Phase %v", p.String())
		}
		if zero.Cmp(actual.Z) != 0 {
			t.Errorf("Test of NewRoundWithPhase() found Round Z is not set to Max4kBitInt for Phase %v", p.String())
		}

		if actual.BatchSize != size {
			t.Errorf("Test of NewRoundWithPhase() found Round BatchSize is not 42 for Phase %v", p.String())
		}

		if actual.GetPhase() != p {
			t.Errorf("Test of NewRoundWithPhase() found Phase is %v instead of %v", actual.GetPhase().String(), p.String())
		}

		for i := uint64(0); i < size; i++ {
			if zero.Cmp(actual.R[i]) != 0 {
				t.Errorf("Test of NewRoundWithPhase() found Round R[%d] is not set to Max4kBitInt for Phase %v", i, p.String())
			}
			if zero.Cmp(actual.S[i]) != 0 {
				t.Errorf("Test of NewRoundWithPhase() found Round S[%d] is not set to Max4kBitInt for Phase %v", i, p.String())
			}
			if zero.Cmp(actual.T[i]) != 0 {
				t.Errorf("Test of NewRoundWithPhase() found Round T[%d] is not set to Max4kBitInt for Phase %v", i, p.String())
			}
			if zero.Cmp(actual.V[i]) != 0 {
				t.Errorf("Test of NewRoundWithPhase() found Round V[%d] is not set to Max4kBitInt for Phase %v", i, p.String())
			}
			if zero.Cmp(actual.U[i]) != 0 {
				t.Errorf("Test of NewRoundWithPhase() found Round U[%d] is not set to Max4kBitInt for Phase %v", i, p.String())
			}
			if zero.Cmp(actual.R_INV[i]) != 0 {
				t.Errorf("Test of NewRoundWithPhase() found Round R_INV[%d] is not set to Max4kBitInt for Phase %v", i, p.String())
			}
			if zero.Cmp(actual.S_INV[i]) != 0 {
				t.Errorf("Test of NewRoundWithPhase() found Round S_INV[%d] is not set to Max4kBitInt for Phase %v", i, p.String())
			}
			if zero.Cmp(actual.T_INV[i]) != 0 {
				t.Errorf("Test of NewRoundWithPhase() found Round T_INV[%d] is not set to Max4kBitInt for Phase %v", i, p.String())
			}
			if zero.Cmp(actual.V_INV[i]) != 0 {
				t.Errorf("Test of NewRoundWithPhase() found Round V_INV[%d] is not set to Max4kBitInt for Phase %v", i, p.String())
			}
			if zero.Cmp(actual.U_INV[i]) != 0 {
				t.Errorf("Test of NewRoundWithPhase() found Round U_INV[%d] is not set to Max4kBitInt for Phase %v", i, p.String())
			}
			if actual.Permutations[i] != i {
				t.Errorf("Test of NewRoundWithPhase() found Round Permutations[%d] is not set to its own permutation for Phase %v", i, p.String())
			}
			if zero.Cmp(actual.Y_R[i]) != 0 {
				t.Errorf("Test of NewRoundWithPhase() found Round Y_R[%d] is not set to Max4kBitInt for Phase %v", i, p.String())
			}
			if zero.Cmp(actual.Y_S[i]) != 0 {
				t.Errorf("Test of NewRoundWithPhase() found Round Y_S[%d] is not set to Max4kBitInt for Phase %v", i, p.String())
			}
			if zero.Cmp(actual.Y_T[i]) != 0 {
				t.Errorf("Test of NewRoundWithPhase() found Round Y_T[%d] is not set to Max4kBitInt for Phase %v", i, p.String())
			}
			if zero.Cmp(actual.Y_V[i]) != 0 {
				t.Errorf("Test of NewRoundWithPhase() found Round Y_V[%d] is not set to Max4kBitInt for Phase %v", i, p.String())
			}
			if zero.Cmp(actual.Y_U[i]) != 0 {
				t.Errorf("Test of NewRoundWithPhase() found Round Y_U[%d] is not set to Max4kBitInt for Phase %v", i, p.String())
			}
		}
	}
}

func TestSetPhase(t *testing.T) {
	round := &Round{}
	round.phaseCond = &sync.Cond{L: &sync.Mutex{}}
	phaseWaitCheck := 0

	go func(r *Round, p *int) {
		r.WaitUntilPhase(REAL_DECRYPT)
		*p = 1
	}(round, &phaseWaitCheck)

	for q := Phase(OFF); q < NUM_PHASES; q++ {
		round.SetPhase(q)
		if round.phase != q {
			t.Errorf("Failed to set phase to %d!", q)
		}
	}
	// Give the goroutine some extra time to run
	time.Sleep(100 * time.Millisecond)
	if phaseWaitCheck != 1 {
		t.Errorf("round.WaitUntilPhase did not complete!")
	}
}

// Test happy path
func TestRoundMap_DeleteRound(t *testing.T) {
	rm := NewRoundMap()
	roundId := "test"
	rm.rounds[roundId] = NewRound(1, Group)
	rm.DeleteRound(roundId)
	if rm.rounds[roundId] != nil {
		t.Errorf("DeleteRound: Failed to delete round!")
	}
}

// Test nil path
func TestRound_GetChannelNil(t *testing.T) {
	var r *Round = nil
	c := r.GetChannel(PRECOMP_REVEAL)
	if c == nil {
		t.Errorf("GetChannel: Expected non-nil return for nil round!")
	}
}
