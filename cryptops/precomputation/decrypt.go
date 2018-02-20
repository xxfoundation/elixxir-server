// Copyright © 2018 Privategrity Corporation
//
// All rights reserved.
//
// Implements the Precomputation Decrypt phase

package precomputation

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
)

// Decrypt phase transforms first unpermuted internode keys
// and partial cypher texts into the data that the permute phase needs
type Decrypt struct{}

// KeysDecrypt holds the keys used by the Decrypt Operation
type KeysDecrypt struct {
	// Public Key for entire round generated in Share Phase
	PublicCypherKey *cyclic.Int
	// Message Public Key Inverse
	R_INV *cyclic.Int
	// Message Private Key
	Y_R *cyclic.Int
	// Recipient Public Key Inverse
	U_INV *cyclic.Int
	// Recipient Private Key
	Y_U *cyclic.Int
}

// Allocated memory and arranges key objects for the Precomputation
// Decrypt Phase
func (d Decrypt) Build(g *cyclic.Group, face interface{}) (
	*services.DispatchBuilder) {

	// Get round from the empty interface
	round := face.(*globals.Round)

	// Allocate Memory for output
	om := make([]services.Slot, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = &PrecomputationSlot{
			Slot:                      i,
			MessagePrecomputation:     cyclic.NewMaxInt(),
			MessageCypher:             cyclic.NewMaxInt(),
			RecipientIDPrecomputation: cyclic.NewMaxInt(),
			RecipientIDCypher:         cyclic.NewMaxInt(),
		}
	}

	keys := make([]services.NodeKeys, round.BatchSize)

	// Link the keys for decryption
	for i := uint64(0); i < round.BatchSize; i++ {
		keySlc := &KeysDecrypt{
			PublicCypherKey: round.CypherPublicKey,
			R_INV:           round.R_INV[i],
			Y_R:             round.Y_R[i],
			U_INV:           round.U_INV[i],
			Y_U:             round.Y_U[i],
		}
		keys[i] = keySlc
	}

	db := services.DispatchBuilder{
		BatchSize: round.BatchSize,
		Keys: &keys,
		Output: &om,
		G: g,
	}

	return &db
}

// Multiplies in own Encrypted Keys and Partial Cypher Texts
func (d Decrypt) Run(g *cyclic.Group, in, out *PrecomputationSlot,
	keys *KeysDecrypt) services.Slot {

	// Create Temporary variable
	tmp := cyclic.NewMaxInt()

	// Eq 12.1: Combine First Unpermuted Internode Message Keys
	g.Exp(g.G, keys.Y_R, tmp)
	g.Mul(keys.R_INV, tmp, tmp)
	g.Mul(in.MessageCypher, tmp, out.MessageCypher)

	// Eq 12.3: Combine First Unpermuted Internode Recipient Keys
	g.Exp(g.G, keys.Y_U, tmp)
	g.Mul(keys.U_INV, tmp, tmp)
	g.Mul(in.RecipientIDCypher, tmp, out.RecipientIDCypher)

	// Eq 12.5: Combine Partial Message Cypher Text
	g.Exp(keys.PublicCypherKey, keys.Y_R, tmp)
	g.Mul(in.MessagePrecomputation, tmp, out.MessagePrecomputation)

	// Eq 12.7: Combine Partial Recipient Cypher Text
	g.Exp(keys.PublicCypherKey, keys.Y_U, tmp)
	g.Mul(in.RecipientIDPrecomputation, tmp, out.RecipientIDPrecomputation)

	return out

}
