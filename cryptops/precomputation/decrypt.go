// Copyright © 2018 Privategrity Corporation
//
// All rights reserved.
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

// SlotDecrypt is used to pass external data into Decrypt and to pass the results out of Decrypt
type SlotDecrypt struct {
	//Slot Number of the Data
	Slot uint64
	// Eq 12.9: First unpermuted internode message key from previous node
	EncryptedMessageKeys *cyclic.Int
	// Eq 12.11: First unpermuted internode recipient ID key from previous node
	EncryptedRecipientIDKeys *cyclic.Int
	// Eq 12.13: Partial cipher test for first unpermuted internode message key from previous node
	PartialMessageCypherText *cyclic.Int
	// Eq 12.15: Partial cipher test for first unpermuted internode recipient ID key from previous node
	PartialRecipientIDCypherText *cyclic.Int
}

// SlotID Returns the Slot number
func (e *SlotDecrypt) SlotID() uint64 {
	return e.Slot
}

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

// Allocated memory and arranges key objects for the Precomputation Decrypt Phase
func (d Decrypt) Build(g *cyclic.Group, face interface{}) *services.DispatchBuilder {

	// Get round from the empty interface
	round := face.(*globals.Round)

	// Allocate Memory for output
	om := make([]services.Slot, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = &SlotDecrypt{
			Slot:                         i,
			EncryptedMessageKeys:         cyclic.NewMaxInt(),
			PartialMessageCypherText:     cyclic.NewMaxInt(),
			EncryptedRecipientIDKeys:     cyclic.NewMaxInt(),
			PartialRecipientIDCypherText: cyclic.NewMaxInt(),
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

	db := services.DispatchBuilder{BatchSize: round.BatchSize, Keys: &keys, Output: &om, G: g}

	return &db

}

// Multiplies in own Encrypted Keys and Partial Cypher Texts
func (d Decrypt) Run(g *cyclic.Group, in, out *SlotDecrypt, keys *KeysDecrypt) services.Slot {

	// Create Temporary variable
	tmp := cyclic.NewMaxInt()

	// Eq 12.1: Combine First Unpermuted Internode Message Keys
	g.Exp(g.G, keys.Y_R, tmp)
	g.Mul(keys.R_INV, tmp, tmp)
	g.Mul(in.EncryptedMessageKeys, tmp, out.EncryptedMessageKeys)

	// Eq 12.3: Combine First Unpermuted Internode Recipient Keys
	g.Exp(g.G, keys.Y_U, tmp)
	g.Mul(keys.U_INV, tmp, tmp)
	g.Mul(in.EncryptedRecipientIDKeys, tmp, out.EncryptedRecipientIDKeys)

	// Eq 12.5: Combine Partial Message Cypher Text
	g.Exp(keys.PublicCypherKey, keys.Y_R, tmp)
	g.Mul(in.PartialMessageCypherText, tmp, out.PartialMessageCypherText)

	// Eq 12.7: Combine Partial Recipient Cypher Text
	g.Exp(keys.PublicCypherKey, keys.Y_U, tmp)
	g.Mul(in.PartialRecipientIDCypherText, tmp, out.PartialRecipientIDCypherText)

	return out

}
