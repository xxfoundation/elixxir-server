package server

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"testing"
)

// TestNewRound tests that the round constructor really only returns
// an empty round with everything initialized to 0.
func TestNewRound(t *testing.T) {
	var actual *round
	size := uint64(42)
	actual = NewRound(size)

	zero := cyclic.NewInt(0)
	zero.SetBytes(Max4192BitInt)

	if (zero.Cmp(actual.G) != 0) {
		t.Errorf("Round G is not set to Max4192BitInt")
	}
	if (zero.Cmp(actual.Z) != 0) {
		t.Errorf("Round Z is not set to Max4192BitInt")
	}
	for i := uint64(0); i < size; i++ {
		if (zero.Cmp(actual.R[i]) != 0) {
			t.Errorf("Round R[%d] is not set to Max4192BitInt", i)
		}
		if (zero.Cmp(actual.S[i]) != 0) {
			t.Errorf("Round S[%d] is not set to Max4192BitInt", i)
		}
		if (zero.Cmp(actual.T[i]) != 0) {
			t.Errorf("Round T[%d] is not set to Max4192BitInt", i)
		}
		if (zero.Cmp(actual.V[i]) != 0) {
			t.Errorf("Round V[%d] is not set to Max4192BitInt", i)
		}
		if (zero.Cmp(actual.U[i]) != 0) {
			t.Errorf("Round U[%d] is not set to Max4192BitInt", i)
		}
		if (actual.Permutations[i] != 0) {
			t.Errorf("Round Permutations[%d] is not set to Max4192BitInt", i)
		}
		if (zero.Cmp(actual.Y_R[i]) != 0) {
			t.Errorf("Round Y_R[%d] is not set to Max4192BitInt", i)
		}
		if (zero.Cmp(actual.Y_S[i]) != 0) {
			t.Errorf("Round Y_S[%d] is not set to Max4192BitInt", i)
		}
		if (zero.Cmp(actual.Y_T[i]) != 0) {
			t.Errorf("Round Y_T[%d] is not set to Max4192BitInt", i)
		}
		if (zero.Cmp(actual.Y_V[i]) != 0) {
			t.Errorf("Round Y_V[%d] is not set to Max4192BitInt", i)
		}
		if (zero.Cmp(actual.Y_U[i]) != 0) {
			t.Errorf("Round Y_U[%d] is not set to Max4192BitInt", i)
		}
	}
}
