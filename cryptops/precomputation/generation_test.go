////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package precomputation

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/services"
	"testing"
)

func TestGeneration(t *testing.T) {
	// NOTE: Does not test correctness.

	test := 1
	pass := 0

	prime := large.NewIntFromString(
		"FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1"+
			"29024E088A67CC74020BBEA63B139B22514A08798E3404DD"+
			"EF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245"+
			"E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7ED"+
			"EE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3D"+
			"C2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F"+
			"83655D23DCA3AD961C62F356208552BB9ED529077096966D"+
			"670C354E4ABC9804F1746C08CA18217C32905E462E36CE3B"+
			"E39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9"+
			"DE2BCBF6955817183995497CEA956AE515D2261898FA0510"+
			"15728E5A8AAAC42DAD33170D04507A33A85521ABDF1CBA64"+
			"ECFB850458DBEF0A8AEA71575D060C7DB3970F85A6E1E4C7"+
			"ABF5AE8CDB0933D71E8C94E04A25619DCEE3D2261AD2EE6B"+
			"F12FFA06D98A0864D87602733EC86A64521F2B18177B200C"+
			"BBE117577A615D6C770988C0BAD946E208E24FA074E5AB31"+
			"43DB5BFCE0FD108E4B82D120A92108011A723C12A787E6D7"+
			"88719A10BDBA5B2699C327186AF4E23C1A946834B6150BDA"+
			"2583E9CA2AD44CE8DBBBC2DB04DE8EF92E8EFC141FBECAA6"+
			"287C59474E6BC05D99B2964FA090C3A2233BA186515BE7ED"+
			"1F612970CEE2D7AFB81BDD762170481CD0069127D5B05AA9"+
			"93B4EA988D8FDDC186FFB7DC90A6C08F4DF435C934063199"+
			"FFFFFFFFFFFFFFFF", 16)

	pSub1 := large.NewInt(0).Sub(prime, large.NewInt(1))

	grp := cyclic.NewGroup(prime, large.NewInt(11), large.NewInt(13))

	batchSize := uint64(4)

	round := globals.NewRound(batchSize, &grp)

	var inMessages []services.Slot

	for i := uint64(0); i < batchSize; i++ {
		inMessages = append(inMessages, &SlotGeneration{Slot: i})
	}

	dc := services.DispatchCryptop(&grp, Generation{}, nil, nil, round)

	testOK := true
	for i := uint64(0); i < batchSize; i++ {
		dc.InChannel <- &(inMessages[i])
		_ = <-dc.OutChannel
	}

	if !round.Z.GetLargeInt().IsCoprime(pSub1) {
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
