// Copyright © 2018 Privategrity Corporation
//
// All rights reserved.
package precomputation

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
)

// Strip phase inverts the Round Private Keys and used to remove the Homomorphic Encryption from
// the Encrypted Message Keys and the Encrypted Recipient Keys, revealing completed precomputation
type Strip struct{}

// KeysStrip holds the keys used by the Strip Operation
type KeysStrip struct {
	// Eq 16.1
	EncryptedMessageKeys *cyclic.Int
	// Eq 16.2
	EncryptedRecipientKeys *cyclic.Int
}

// Allocated memory and arranges key objects for the Precomputation Strip Phase
func (s Strip) Build(g *cyclic.Group, face interface{}) *services.DispatchBuilder {

	// Get round from the empty interface
	round := face.(*globals.Round)

	// Allocate Memory for output
	om := make([]services.Slot, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		// Attach LastNode to SlotStripOut
		om[i] = &PrecomputationSlot{
			Slot: i,
			MessagePrecomputation:     round.LastNode.MessagePrecomputation[i],
			RecipientIDPrecomputation: round.LastNode.RecipientPrecomputation[i],
		}
	}

	keys := make([]services.NodeKeys, round.BatchSize)

	// Link the keys for stripping
	for i := uint64(0); i < round.BatchSize; i++ {
		keySlc := &KeysStrip{
			EncryptedMessageKeys:   round.LastNode.EncryptedMessagePrecomputation[i],
			EncryptedRecipientKeys: round.LastNode.EncryptedRecipientPrecomputation[i],
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

// Remove Homomorphic Encryption to reveal the Message and Recipient Precomputation
func (s Strip) Run(g *cyclic.Group, in, out *PrecomputationSlot,
	keys *KeysStrip) services.Slot {

	// Create Temporary variable
	tmp := cyclic.NewMaxInt()

	// Eq 16.1: Invert the round message private key
	g.Inverse(in.MessagePrecomputation, tmp)

	// Eq 16.1: Use the inverted round message private key to remove the
	//          homomorphic encryption from encrypted message key and reveal
	//          the message precomputation
	g.Mul(tmp, keys.EncryptedMessageKeys, out.MessagePrecomputation)

	//fmt.Printf("EncryptedRecipientKeys: %s \n",
	//           keys.EncryptedRecipientKeys.Text(10))

	// Eq 16.2: Invert the round recipient private key
	g.Inverse(in.RecipientIDPrecomputation, tmp)

	// Eq 16.2: Use the inverted round recipient private key to remove
	//          the homomorphic encryption from encrypted recipient key and reveal
	//          the recipient precomputation
	g.Mul(tmp, keys.EncryptedRecipientKeys, out.RecipientIDPrecomputation)

	out.MessageCypher = in.MessageCypher
	out.RecipientIDCypher = in.RecipientIDCypher

	return out
}
