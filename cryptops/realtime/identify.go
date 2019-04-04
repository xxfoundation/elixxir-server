////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Implements the Realtime Identify phase

package realtime

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/services"
)

// Identify implements the Identify phase of the realtime processing.
// It removes the keys U and V that encrypt the AssociatedData, so that we can
// start sending the ciphertext to the correct recipient.
type Identify struct{}

// KeysIdentify holds the key needed for the realtime Identify phase
type KeysIdentify struct {
	// Result of the precomputation for the AssociatedData
	// One of the two results of the precomputation
	AssociatedDataPrecomputation *cyclic.Int
}

// Pre-allocate memory and arrange key objects for realtime Identify phase
func (i Identify) Build(grp *cyclic.Group,
	face interface{}) *services.DispatchBuilder {

	// The empty interface should be castable to a Round
	round := face.(*globals.Round)

	// Allocate messages for output
	om := make([]services.Slot, round.BatchSize)

	for j := uint64(0); j < round.BatchSize; j++ {
		om[j] = &Slot{
			Slot:           j,
			AssociatedData: grp.NewMaxInt(),
		}
	}

	keys := make([]services.NodeKeys, round.BatchSize)

	// Prepare the correct keys
	for j := uint64(0); j < round.BatchSize; j++ {
		keySlc := &KeysIdentify{
			AssociatedDataPrecomputation: round.AssociatedDataPrecomputation[j],
		}
		keys[j] = keySlc
	}

	db := services.DispatchBuilder{
		BatchSize: round.BatchSize,
		Keys:      &keys,
		Output:    &om, G: grp}

	return &db
}

// Input: Encrypted AssociatedData, from Permute phase
// This phase decrypts the AssociatedData, making it possible to identify the recipient
func (i Identify) Run(g *cyclic.Group,
	in, out *Slot, keys *KeysIdentify) services.Slot {

	// Eq 5.1
	// Multiply AssociatedData by the precomputed value
	g.Mul(in.AssociatedData, keys.AssociatedDataPrecomputation,
		out.AssociatedData)

	return out
}
