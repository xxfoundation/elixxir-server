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
	// AssociatedData Public Key Inverse
	U_INV *cyclic.Int
	// AssociatedData Private Key
	Y_U *cyclic.Int
}

// Allocated memory and arranges key objects for the Precomputation
// Decrypt Phase
func (d Decrypt) Build(grp *cyclic.Group, face interface{}) *services.DispatchBuilder {

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
		Keys:      &keys,
		Output:    &om,
		G:         grp,
	}

	return &db
}

// Multiplies in own Encrypted Keys and Partial Cypher Texts
func (d Decrypt) Run(grp *cyclic.Group, in, out *PrecomputationSlot,
	keys *KeysDecrypt) services.Slot {

	// Create Temporary variable
	tmp := grp.NewMaxInt()

	// Eq 12.1: Combine First Unpermuted Internode Message Keys
	grp.Exp(grp.GetGCyclic(), keys.Y_R, tmp)
	grp.Mul(keys.R_INV, tmp, tmp)
	grp.Mul(in.MessageCypher, tmp, out.MessageCypher)

	// Eq 12.3: Combine First Unpermuted Internode AssociatedData Keys
	grp.Exp(grp.GetGCyclic(), keys.Y_U, tmp)
	grp.Mul(keys.U_INV, tmp, tmp)
	grp.Mul(in.AssociatedDataCypher, tmp, out.AssociatedDataCypher)

	// Eq 12.5: Combine Partial Message Cypher Text
	grp.Exp(keys.PublicCypherKey, keys.Y_R, tmp)
	grp.Mul(in.MessagePrecomputation, tmp, out.MessagePrecomputation)

	// Eq 12.7: Combine Partial AssociatedData Cypher Text
	grp.Exp(keys.PublicCypherKey, keys.Y_U, tmp)
	grp.Mul(in.AssociatedDataPrecomputation, tmp, out.AssociatedDataPrecomputation)

	return out

}
