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
)

// Reveal implements the Reveal phase of the precomputation. It removes the
// cypher keys from the message and associated data cypher text, revealing the
// private keys for the round
type Reveal struct{}

// KeysReveal holds the keys used by the Reveal operation
type KeysReveal struct {
	// Private cypher key for all messages in the round
	// Generated in Generation phase
	Z *cyclic.Int
}

// Pre-allocate memory and arrange key objects for Precomputation Reveal phase
func (r Reveal) Build(grp *cyclic.Group, face interface{}) *services.DispatchBuilder {

	// The empty interface should be able to be casted to a Round
	round := face.(*globals.Round)

	// Allocate messages for output
	om := make([]services.Slot, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = &PrecomputationSlot{
			Slot:                         i,
			MessagePrecomputation:        grp.NewMaxInt(),
			AssociatedDataPrecomputation: grp.NewMaxInt(),
		}
	}

	keys := make([]services.NodeKeys, round.BatchSize)

	// Prepare the correct keys
	for i := uint64(0); i < round.BatchSize; i++ {
		keySlc := &KeysReveal{Z: round.Z}
		keys[i] = keySlc
	}

	db := services.DispatchBuilder{
		BatchSize: round.BatchSize,
		Keys:      &keys,
		Output:    &om,
		G:         grp,
	}

	return &db
}

// Input: Partial message cypher text, from Encrypt Phase
//        Partial associated data cypher text, from Permute Phase
// This phase removes the homomorphic encryption from these two quantities.
func (r Reveal) Run(g *cyclic.Group, in, out *PrecomputationSlot,
	keys *KeysReveal) services.Slot {

	// Eq 15.11 Root by cypher key to remove one layer of homomorphic
	// encryption from partially encrypted message cypher text.
	g.RootCoprime(in.MessagePrecomputation, keys.Z, out.MessagePrecomputation)

	// Eq 15.13 Root by cypher key to remove one layer of homomorphic
	// encryption from partially encrypted associated data cypher text.
	g.RootCoprime(in.AssociatedDataPrecomputation, keys.Z,
		out.AssociatedDataPrecomputation)

	out.MessageCypher = in.MessageCypher
	out.AssociatedDataCypher = in.AssociatedDataCypher

	return out
}
