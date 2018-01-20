// Implements the Precomputation Reveal phase

package precomputation

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/node"
	"gitlab.com/privategrity/server/services"
)

// Reveal implements the Reveal phase of the precomputation. It removes the
// cypher keys from the message and recipient cypher text, revealing the
// private keys for the round
type Reveal struct{}

// (r Reveal) Run() uses SlotReveal structs to pass data into and out of Reveal
type SlotReveal struct {
	// Slot number
	slot uint64

	// Partially decrypted message cypher texts
	PartialMessageCypherText *cyclic.Int
	// Partially decrypted recipient cypher texts
	PartialRecipientCypherText *cyclic.Int
}

// SlotID() gets the slot number
func (r *SlotReveal) SlotID() uint64 {
	return r.slot
}

// KeysReveal holds the keys used by the Reveal operation
type KeysReveal struct {
	// Private cypher key for all messages in the round
	// Generated in Generation phase
	PrivateCypherKey *cyclic.Int
}

// Pre-allocate memory and arrange key objects for Precomputation Reveal phase
func (r Reveal) Build(g *cyclic.Group, face interface{}) *services.DispatchBuilder {
	// The empty interface should be castable to a Round
	round := face.(*node.Round)

	// Allocate messages for output
	om := make([]services.Slot, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = &SlotReveal{slot: i,
			PartialMessageCypherText:   cyclic.NewMaxInt(),
			PartialRecipientCypherText: cyclic.NewMaxInt(),
		}
	}

	keys := make([]services.NodeKeys, round.BatchSize)

	// Prepare the correct keys
	for i := uint64(0); i < round.BatchSize; i++ {
		keySlc := &KeysReveal{PrivateCypherKey: round.Z}
		keys[i] = keySlc
	}

	db := services.DispatchBuilder{BatchSize: round.BatchSize, Keys: &keys, Output: &om, G: g}

	return &db
}

func (r Reveal) Run(g *cyclic.Group, in, out *SlotReveal,
	keys *KeysReveal) services.Slot {
	// Input: Partial message cypher text, from Encrypt Phase
	//        Partial recipient ID cypher text, from Permute Phase
	// This phase removes the homomorphic encryption from these two quantities.

	// Root by cypher key to remove one layer of homomorphic encryption
	// from partially encrypted message cypher text. Eq 15.11
	g.Root(in.PartialMessageCypherText, keys.PrivateCypherKey,
		out.PartialMessageCypherText)

	// Root by cypher key to remove one layer of homomorphic encryption
	// from partially encrypted recipient ID cypher text. Eq 15.13
	g.Root(in.PartialRecipientCypherText, keys.PrivateCypherKey,
		out.PartialRecipientCypherText)

	return out
}
