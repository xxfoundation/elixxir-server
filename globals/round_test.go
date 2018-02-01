package globals

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"sync"
	"testing"
)

// TestNewRound tests that the round constructor really only returns
// an empty round with everything initialized to 0.
func TestNewRound(t *testing.T) {
	var actual *Round
	size := uint64(42)
	actual = NewRound(size)

	zero := cyclic.NewInt(0)
	zero.SetBytes(cyclic.Max4kBitInt)

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
	round.phaseLock = &sync.Mutex{}

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
		actual = NewRoundWithPhase(size, p)

		zero := cyclic.NewInt(0)
		zero.SetBytes(cyclic.Max4kBitInt)

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

func TestIncrementPhase(t *testing.T) {
	round := &Round{}
	round.phaseLock = &sync.Mutex{}

	for p := OFF; p < DONE; p++ {
		for q := OFF; q < NUM_PHASES; q++ {
			round.phase = p

			err := round.IncrementPhase(q)

			if (p+1 != q) && err == nil {
				t.Errorf("Test of IncrementPhase() did not find an error on Phase %v which tried to increment to  %v", p.String(), q.String())
			}

			if (p+1 == q) && err != nil {
				t.Errorf("Test of IncrementPhase() found an error on Phase %v when trying to increment to phase %v which read: %v", p.String(), q.String(), err.Error())
			}

		}
	}

	round.phase = DONE

	for p := OFF; p < NUM_PHASES; p++ {
		err := round.IncrementPhase(p)

		if err == nil {
			t.Errorf("Test of IncrementPhase() on DONE did not find an error when incrementing to Phase %v", p.String())
		}

	}

	round.phase = ERROR

	for p := OFF; p < NUM_PHASES; p++ {
		err := round.IncrementPhase(p)

		if err == nil {
			t.Errorf("Test of IncrementPhase() on ERROR did not find an error when incrementing to Phase %v", p.String())
		}

	}

}

func TestError(t *testing.T) {
	round := &Round{}
	round.phaseLock = &sync.Mutex{}

	for p := OFF; p < NUM_PHASES; p++ {
		round.phase = p

		round.Error()

		if round.phase != ERROR {
			t.Errorf("Test of Error() failed on Phase %v", p.String())
		}
	}
}
