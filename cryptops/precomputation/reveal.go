// Implements the Precomputation Reveal phase

package precomputation

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/node"
	"gitlab.com/privategrity/server/services"
)

// Reveal phase removes Homomorphic Encryption from the Cypher Texts to reveal the Round Private Key
type Reveal struct{}

// SlotReveal is used to pass external data into Reveal and to pass the results out of Reveal
type SlotReveal struct {
	//Slot Number of the Data
	slot uint64
	// Partially Decrypted Message Cypher Text
	MessageCypherText *cyclic.Int
	// Partially Decrypted Recipient Cypher Text
	RecipientCypherText *cyclic.Int
}

// SlotID Returns the Slot number
func (e *SlotReveal) SlotID() uint64 {
	return e.slot
}

// KeysReveal holds the keys used by the Reveal Operation
type KeysReveal struct {
	// Private Cypher Key
	Z *cyclic.Int
}

// Allocated memory and arranges key objects for the Precomputation Reveal Phase
func (r Reveal) Build(g *cyclic.Group, face interface{}) *services.DispatchBuilder {

	// Get round from the empty interface
	round := face.(*node.Round)

	// Allocate Memory for output
	om := make([]services.Slot, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = &SlotReveal{
			slot:                i,
			MessageCypherText:   cyclic.NewMaxInt(),
			RecipientCypherText: cyclic.NewMaxInt(),
		}
	}

	keys := make([]services.NodeKeys, round.BatchSize)

	// Link the keys for reveal
	for i := uint64(0); i < round.BatchSize; i++ {
		keySlc := &KeysReveal{
			Z: round.Z,
		}
		keys[i] = keySlc
	}

	db := services.DispatchBuilder{BatchSize: round.BatchSize, Keys: &keys, Output: &om, G: g}

	return &db

}

// Root the cypher texts by the Private Cypher Key to reveal the Round Private Key
func (r Reveal) Run(g *cyclic.Group, in, out *SlotReveal, keys *KeysReveal) services.Slot {

	// Eq 15.11: Root MessageCypherText by Private Cypher Key
	g.Root(in.MessageCypherText, keys.Z, out.MessageCypherText)
	// Eq 15.13: Root RecipientCypherText by Private Cypher Key
	g.Root(in.RecipientCypherText, keys.Z, out.RecipientCypherText)

	return out

}
