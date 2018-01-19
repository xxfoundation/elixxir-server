// Implements the Precomputation Share phase

package precomputation

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/node"
	"gitlab.com/privategrity/server/services"
)

// Share phase generates the Round Public Cypher Key to share with all nodes
type Share struct{}

// SlotShare is used to pass external data into Share and to pass the results out of Share
type SlotShare struct {

	//Slot Number of the Data
	slot uint64

	// Eq 10.3: Partial result of raising the global generator to the power of each node's Private Cypher Key
	PartialRoundPublicCypherKey *cyclic.Int
}

// SlotID Returns the Slot number
func (e *SlotShare) SlotID() uint64 {
	return e.slot
}

// KeysShare holds the keys used by the Share Operation
type KeysShare struct {

	// Private Cypher Key
	PrivateCypherKey *cyclic.Int
}

// Allocated memory and arranges key objects for the Precomputation Share Phase
func (s Share) Build(g *cyclic.Group, face interface{}) *services.DispatchBuilder {

	// Get round from the empty interface
	round := face.(*node.Round)

	// Allocate Memory for output
	om := make([]services.Slot, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = &SlotShare{slot: i,
			PartialRoundPublicCypherKey: cyclic.NewMaxInt()}
	}

	keys := make([]services.NodeKeys, round.BatchSize)

	// Link the keys for sharing
	for i := uint64(0); i < round.BatchSize; i++ {
		keySlc := &KeysShare{PrivateCypherKey: round.Z}
		keys = append(keys, keySlc)
	}

	db := services.DispatchBuilder{BatchSize: round.BatchSize, Keys: &keys, Output: &om, G: g}

	return &db

}

// Partial Public Cypher Key is passed from node to node, each raising it
// to the power of its Private Cypher Key in order to generate the Round Public Cypher Key
func (s Share) Run(g *cyclic.Group, in, out *SlotShare, keys *KeysShare) services.Slot {

	// Eq 10.4: Raise PartialRoundPublicCypherKey to the power of own Private Cypher Key
	g.Exp(in.PartialRoundPublicCypherKey, keys.PrivateCypherKey, out.PartialRoundPublicCypherKey)
	return out

}
