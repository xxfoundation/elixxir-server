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

// Permute phase permutes the message keys, the associated data keys, and their cypher
// text, while multiplying in its own keys.
type Permute struct{}

// KeysPermute holds the keys used by the Permute Operation
type KeysPermute struct {
	// Public Key for entire round generated in Share Phase
	CypherPublicKey *cyclic.Int
	// Encrypted Inverse Permuted Internode Message Key
	S_INV *cyclic.Int
	// Encrypted Inverse Permuted Internode AssociatedData Key
	V_INV *cyclic.Int
	// Permuted Internode Message Partial Cypher Text
	Y_S *cyclic.Int
	// Permuted Internode AssociatedData Partial Cypher Text
	Y_V *cyclic.Int
}

// Allocated memory and arranges key objects for the Precomputation
// Permute Phase
func (p Permute) Build(grp *cyclic.Group, face interface{}) *services.DispatchBuilder {

	// Get round from the empty interface
	round := face.(*globals.Round)

	// Allocate Memory for output
	om := make([]services.Slot, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = &PrecomputationSlot{
			Slot:                         i,
			MessagePrecomputation:        grp.NewMaxInt(),
			MessageCypher:                grp.NewMaxInt(),
			AssociatedDataPrecomputation: grp.NewMaxInt(),
			AssociatedDataCypher:         grp.NewMaxInt(),
		}
	}

	buildCryptoPermute(round, &om)

	keys := make([]services.NodeKeys, round.BatchSize)

	// Link the keys for permutation
	for i := uint64(0); i < round.BatchSize; i++ {
		keySlc := &KeysPermute{
			CypherPublicKey: round.CypherPublicKey,
			S_INV:           round.S_INV[i],
			V_INV:           round.V_INV[i],
			Y_S:             round.Y_S[i],
			Y_V:             round.Y_V[i],
		}
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

// Permutes the four results of the Decrypt phase and multiplies in own keys
func (p Permute) Run(grp *cyclic.Group, in, out *PrecomputationSlot,
	keys *KeysPermute) services.Slot {

	// Create Temporary variable
	tmp := grp.NewMaxInt()

	// Eq 11.1: Encrypt the Permuted Internode Message Key under
	// Homomorphic Encryption
	grp.Exp(grp.GetGCyclic(), keys.Y_S, tmp)
	grp.Mul(keys.S_INV, tmp, tmp)

	// Eq 13.17: Multiplies the Permuted Internode Message Key under Homomorphic
	// Encryption into the Partial Encrypted Message Precomputation
	grp.Mul(in.MessageCypher, tmp, out.MessageCypher)

	// Eq 11.1: Encrypt the Permuted Internode AssociatedData Key under Homomorphic
	// Encryption
	grp.Exp(grp.GetGCyclic(), keys.Y_V, tmp)
	grp.Mul(keys.V_INV, tmp, tmp)

	// Eq 13.19: Multiplies the Permuted Internode AssociatedData Key under
	// Homomorphic Encryption into the Partial Encrypted AssociatedData Precomputation
	grp.Mul(in.AssociatedDataCypher, tmp, out.AssociatedDataCypher)

	// Eq 11.2: Makes the Partial Cypher Text for the Permuted Internode Message
	// Key
	grp.Exp(keys.CypherPublicKey, keys.Y_S, tmp)

	// Eq 13.21: Multiplies the Partial Cypher Text for the Permuted Internode
	// Message Key into the Round Message Partial Cypher Text
	grp.Mul(in.MessagePrecomputation, tmp, out.MessagePrecomputation)

	// Eq 11.2: Makes the Partial Cypher Text for the Permuted Internode
	// AssociatedData Key
	grp.Exp(keys.CypherPublicKey, keys.Y_V, tmp)

	// Eq 13.23 Multiplies the Partial Cypher Text for the Permuted Internode
	// AssociatedData Key into the Round AssociatedData Partial Cypher Text
	grp.Mul(in.AssociatedDataPrecomputation, tmp, out.AssociatedDataPrecomputation)

	return out

}

// Implements cryptographic component of build
func buildCryptoPermute(round *globals.Round, om *[]services.Slot) {
	// Set the Slot to the respective permutation
	for i := uint64(0); i < round.BatchSize; i++ {
		(*om)[i].(*PrecomputationSlot).Slot = round.Permutations[i]
	}
}
