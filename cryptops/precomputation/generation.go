////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package precomputation

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/services"
)

// Generation phase generates all the keys used in the encryption.
type Generation struct{}

// KeysGeneration holds the keys used by the Generation Operation
type KeysGeneration struct {
	R     *cyclic.Int
	S     *cyclic.Int
	T     *cyclic.Int
	U     *cyclic.Int
	V     *cyclic.Int
	R_INV *cyclic.Int
	S_INV *cyclic.Int
	T_INV *cyclic.Int
	U_INV *cyclic.Int
	V_INV *cyclic.Int
	Y_R   *cyclic.Int
	Y_S   *cyclic.Int
	Y_T   *cyclic.Int
	Y_U   *cyclic.Int
	Y_V   *cyclic.Int
}

// Allocated memory and arranges key objects for the Precomputation
// Generation Phase
func (gen Generation) Build(g *cyclic.Group, face interface{}) *services.DispatchBuilder {

	// Get round from the empty interface
	round := face.(*globals.Round)

	// Create the permutation and generate a Private Cypher Key
	buildCryptoGeneration(g, round)

	// Allocate Memory for output
	om := make([]services.Slot, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = &SlotGeneration{
			Slot: i,
		}
	}

	keys := make([]services.NodeKeys, round.BatchSize)

	// Link the keys for generation
	for i := uint64(0); i < round.BatchSize; i++ {
		keySlc := &KeysGeneration{
			R:     round.R[i],
			S:     round.S[i],
			T:     round.T[i],
			U:     round.U[i],
			V:     round.V[i],
			R_INV: round.R_INV[i],
			S_INV: round.S_INV[i],
			T_INV: round.T_INV[i],
			U_INV: round.U_INV[i],
			V_INV: round.V_INV[i],
			Y_R:   round.Y_R[i],
			Y_S:   round.Y_S[i],
			Y_T:   round.Y_T[i],
			Y_U:   round.Y_U[i],
			Y_V:   round.Y_V[i],
		}
		keys[i] = keySlc
	}

	db := services.DispatchBuilder{
		BatchSize: round.BatchSize,
		Keys:      &keys,
		Output:    &om,
		G:         g,
	}

	return &db

}

// Generate random values for all keys
func (gen Generation) Run(g *cyclic.Group, in, out *SlotGeneration,
	keys *KeysGeneration) services.Slot {

	// Generates a random value within the group for every internode key
	g.Random(keys.R)
	g.Random(keys.S)
	g.Random(keys.T)
	g.Random(keys.U)
	g.Random(keys.V)

	// Generates the inverse keys
	g.Inverse(keys.R, keys.R_INV)
	g.Inverse(keys.S, keys.S_INV)
	g.Inverse(keys.T, keys.T_INV)
	g.Inverse(keys.U, keys.U_INV)
	g.Inverse(keys.V, keys.V_INV)

	// Generates a random value within the group for every private key

	csprig := csprng.NewSystemRNG()

	yr := make([]byte, 32)
	_, err := csprig.Read(yr)

	if err != nil {
		jww.FATAL.Panicf("Count not generate key yr: %s", err.Error())
	}
	g.SetBytes(keys.Y_R, yr)

	ys := make([]byte, 32)
	_, err = csprig.Read(ys)

	if err != nil {
		jww.FATAL.Panicf("Count not generate key ys: %s", err.Error())
	}
	g.SetBytes(keys.Y_S, ys)

	yt := make([]byte, 32)
	_, err = csprig.Read(yt)

	if err != nil {
		jww.FATAL.Panicf("Count not generate key yt: %s", err.Error())
	}
	g.SetBytes(keys.Y_T, yt)

	yu := make([]byte, 32)
	_, err = csprig.Read(yu)

	if err != nil {
		jww.FATAL.Panicf("Count not generate key: %s yu", err.Error())
	}
	g.SetBytes(keys.Y_U, yu)

	yv := make([]byte, 16)
	_, err = csprig.Read(yv)

	if err != nil {
		jww.FATAL.Panicf("Count not generate key: %s yv", err.Error())
	}
	g.SetBytes(keys.Y_V, yv)

	return out

}

// Implements cryptographic component of build
func buildCryptoGeneration(g *cyclic.Group, round *globals.Round) {

	// Make the Permutation
	cyclic.Shuffle(&round.Permutations)

	// Generate the Private Cypher Key
	g.FindSmallCoprimeInverse(round.Z, round.ExpSize)

}
