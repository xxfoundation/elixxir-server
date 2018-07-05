////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Implements the Realtime Permute phase

package realtime

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
)

// The realtime Permute phase blindly permutes, then re-encrypts messages
// and recipient IDs to prevent other nodes or outside actors from knowing
// the origin of the messages.
type Permute struct{}

// KeysPermute holds the keys used by the Permute operation
type KeysPermute struct {
	S *cyclic.Int
	V *cyclic.Int
}

// Pre-allocate memory and arrange key objects for Realtime Permute phase
func (p Permute) Build(g *cyclic.Group,
	face interface{}) *services.DispatchBuilder {
	// The empty interface should be castable to a Round
	round := face.(*globals.Round)

	// Allocate messages for output
	om := make([]services.Slot, round.BatchSize)

	// BEGIN CRYPTOGRAPHIC PORTION OF BUILD
	buildCryptoPermute(round, om)
	// END CRYPTOGRAPHIC PORTION OF BUILD

	keys := make([]services.NodeKeys, round.BatchSize)

	// Prepare the correct keys
	for i := uint64(0); i < round.BatchSize; i++ {
		keySlc := &KeysPermute{S: round.S[i], V: round.V[i]}

		keys[i] = keySlc
	}

	db := services.DispatchBuilder{BatchSize: round.BatchSize, Keys: &keys,
		Output: &om, G: g}

	return &db
}

// Input: Encrypted message, from Decrypt Phase
//        Encrypted recipient ID, from Decrypt Phase
// This phase permutes the message and the recipient ID and encrypts
// them with their respective permuted internode keys.
func (p Permute) Run(g *cyclic.Group, in, out *Slot,
	keys *KeysPermute) services.Slot {

	// Eq 4.10 Multiply the message by its permuted key to make the permutation
	// secret to the previous node
	g.Mul(in.Message, keys.S, out.Message)

	// Eq 4.12 Multiply the recipient ID by
	// its permuted key making the permutation
	// secret to the previous node
	g.Mul(in.EncryptedRecipient, keys.V, out.EncryptedRecipient)

	return out
}

func buildCryptoPermute(round *globals.Round, outMessages []services.Slot) {
	// Prepare the permuted output messages
	for i := uint64(0); i < round.BatchSize; i++ {
		slot := &Slot{
			Slot:               round.Permutations[i],
			Message:            cyclic.NewMaxInt(),
			EncryptedRecipient: cyclic.NewMaxInt(),
		}
		// If this is the last node, save the EncryptedMessage
		if round.LastNode.EncryptedMessage != nil {
			slot.Message = round.LastNode.EncryptedMessage[i]
		}
		outMessages[i] = slot
	}
}
