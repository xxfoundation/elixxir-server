////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package precomputation

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/services"
	"testing"
)

func TestGeneration(t *testing.T) {
	// NOTE: Does not test correctness.

	test := 1
	pass := 0

	batchSize := uint64(4)

	round := globals.NewRound(batchSize)

	rng := cyclic.NewRandom(cyclic.NewInt(2), cyclic.NewInt(897879897))

	prime := cyclic.NewInt(50581)

	pSub1 := cyclic.NewInt(0).Sub(prime, cyclic.NewInt(1))

	group := cyclic.NewGroup(prime, cyclic.NewInt(11),
		cyclic.NewInt(13), rng)

	var inMessages []services.Slot

	for i := uint64(0); i < batchSize; i++ {
		inMessages = append(inMessages, &SlotGeneration{Slot: i})
	}

	dc := services.DispatchCryptop(&group, Generation{}, nil, nil, round)

	testOK := true
	for i := uint64(0); i < batchSize; i++ {
		dc.InChannel <- &(inMessages[i])
		_ = <-dc.OutChannel
	}

	if !round.Z.IsCoprime(pSub1) {
		t.Errorf("Generation did not generate a coprime for Z, received: %v", round.Z.Text(10))
		testOK = false
	}

	// Only the most basic test of randomness is happening here
	for i := uint64(0); i < batchSize-1; i++ {
		if round.R[i].Cmp(round.R[i+1]) == 0 {
			t.Errorf("Generation generated the same R between slots %d and %d\n", i, i+1)
			testOK = false
		}
		if round.S[i].Cmp(round.S[i+1]) == 0 {
			t.Errorf("Generation generated the same S between slots %d and %d\n", i, i+1)
			testOK = false
		}
		if round.T[i].Cmp(round.T[i+1]) == 0 {
			t.Errorf("Generation generated the same T between slots %d and %d\n", i, i+1)
			testOK = false
		}
		if round.U[i].Cmp(round.U[i+1]) == 0 {
			t.Errorf("Generation generated the same U between slots %d and %d\n", i, i+1)
			testOK = false
		}
		if round.V[i].Cmp(round.V[i+1]) == 0 {
			t.Errorf("Generation generated the same V between slots %d and %d\n", i, i+1)
			testOK = false
		}
		if round.R_INV[i].Cmp(round.R_INV[i+1]) == 0 {
			t.Errorf("Generation generated the same R_INV between slots %d and %d\n", i, i+1)
			testOK = false
		}
		if round.S_INV[i].Cmp(round.S_INV[i+1]) == 0 {
			t.Errorf("Generation generated the same S_INV between slots %d and %d\n", i, i+1)
			testOK = false
		}
		if round.T_INV[i].Cmp(round.T_INV[i+1]) == 0 {
			t.Errorf("Generation generated the same T_INV between slots %d and %d\n", i, i+1)
			testOK = false
		}
		if round.U_INV[i].Cmp(round.U_INV[i+1]) == 0 {
			t.Errorf("Generation generated the same U_INV between slots %d and %d\n", i, i+1)
			testOK = false
		}
		if round.V_INV[i].Cmp(round.V_INV[i+1]) == 0 {
			t.Errorf("Generation generated the same V_INV between slots %d and %d\n", i, i+1)
			testOK = false
		}
		if round.Y_R[i].Cmp(round.Y_R[i+1]) == 0 {
			t.Errorf("Generation generated the same Y_R between slots %d and %d\n", i, i+1)
			testOK = false
		}
		if round.Y_S[i].Cmp(round.Y_S[i+1]) == 0 {
			t.Errorf("Generation generated the same Y_S between slots %d and %d\n", i, i+1)
			testOK = false
		}
		if round.Y_T[i].Cmp(round.Y_T[i+1]) == 0 {
			t.Errorf("Generation generated the same Y_T between slots %d and %d\n", i, i+1)
			testOK = false
		}
		if round.Y_U[i].Cmp(round.Y_U[i+1]) == 0 {
			t.Errorf("Generation generated the same Y_U between slots %d and %d\n", i, i+1)
			testOK = false
		}
		if round.Y_V[i].Cmp(round.Y_V[i+1]) == 0 {
			t.Errorf("Generation generated the same Y_V between slots %d and %d\n", i, i+1)
			testOK = false
		}
	}
	if testOK {
		pass++
	}

	println("Precomputation Generation", pass, "out of", test, "tests "+
		"passed.")
}
