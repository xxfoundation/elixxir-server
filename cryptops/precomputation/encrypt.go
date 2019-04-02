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

// Encrypt implements the Encrypt Phase of the Precomputation. It adds the
// Seconds Unpermuted Internode Message Keys into the Message Precomputation
type Encrypt struct{}

// KeysEncrypt holds the keys used by the Encrypt Operation
type KeysEncrypt struct {
	// Public Key for entire round generated in Share Phase
	CypherPublicKey *cyclic.Int
	// Inverse Second Unpermuted Internode Message Key
	T_INV *cyclic.Int
	// Second Unpermuted Internode Message Private Key
	Y_T *cyclic.Int
}

// Allocated memory and arranges key objects for the Precomputation Encrypt Phase
func (e Encrypt) Build(grp *cyclic.Group, face interface{}) *services.DispatchBuilder {

	// Get round from the empty interface
	round := face.(*globals.Round)

	// Allocate Memory for output
	om := make([]services.Slot, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = &PrecomputationSlot{
			Slot:                  i,
			MessageCypher:         grp.NewMaxInt(),
			MessagePrecomputation: grp.NewMaxInt(),
		}
	}

	keys := make([]services.NodeKeys, round.BatchSize)

	// Link the keys for encryption
	for i := uint64(0); i < round.BatchSize; i++ {
		keySlc := &KeysEncrypt{
			CypherPublicKey: round.CypherPublicKey,
			T_INV:           round.T_INV[i],
			Y_T:             round.Y_T[i],
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

// Output of the Permute Phase is passed to the first Node which
// multiplies in its Encrypted
// Second Unpermuted Message Keys and the associated Private Keys
// into the Partial Message Cypher Test.
func (e Encrypt) Run(grp *cyclic.Group, in, out *PrecomputationSlot,
	keys *KeysEncrypt) services.Slot {

	// Create Temporary variable
	tmp := grp.NewMaxInt()

	// Eq 11.1: Homomorphically encrypt the Inverse Second Unpermuted
	//          Internode Message Key.
	grp.Exp(grp.GetGCyclic(), keys.Y_T, tmp)
	grp.Mul(keys.T_INV, tmp, tmp)

	// Eq 14.5: Multiply the encrypted Inverse Second Unpermuted
	//          Internode Message Key into the partial precomputation.
	grp.Mul(in.MessageCypher, tmp, out.MessageCypher)

	// Eq 11.2: Create the Inverse Second Unpermuted Internode Message Public Key.
	grp.Exp(keys.CypherPublicKey, keys.Y_T, tmp)

	// Multiply cypher the Inverse Second Unpermuted Internode Message
	// Public Key into the Partial Cypher Text.
	grp.Mul(in.MessagePrecomputation, tmp, out.MessagePrecomputation)

	out.AssociatedDataCypher = in.AssociatedDataCypher
	out.AssociatedDataPrecomputation = in.AssociatedDataPrecomputation

	return out

}
