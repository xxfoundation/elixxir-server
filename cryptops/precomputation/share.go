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

// Share phase generates the Round Public Cypher Key to share with all nodes
type Share struct{}

// KeysShare holds the keys used by the Share Operation
type KeysShare struct {
	// Private Cypher Key
	Z *cyclic.Int
}

// Allocated memory and arranges key objects for the Precomputation Share Phase
func (s Share) Build(grp *cyclic.Group, face interface{}) *services.DispatchBuilder {

	// Get round from the empty interface
	round := face.(*globals.Round)

	// Allocate Memory for output
	om := make([]services.Slot, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = &SlotShare{
			Slot:                        i,
			PartialRoundPublicCypherKey: grp.NewMaxInt(),
		}
	}

	keys := make([]services.NodeKeys, round.BatchSize)

	// Link the keys for sharing
	for i := uint64(0); i < round.BatchSize; i++ {
		keySlc := &KeysShare{
			Z: round.Z,
		}
		keys[i] = keySlc
	}

	db := services.DispatchBuilder{
		BatchSize: round.BatchSize, Keys: &keys, Output: &om, G: grp,
	}

	return &db
}

// Partial Public Cypher Key is passed from node to node, each raising it
// to the power of its Private Cypher Key in order to generate the Round Public Cypher Key
func (s Share) Run(g *cyclic.Group, in, out *SlotShare, keys *KeysShare) services.Slot {

	// Eq 10.4: Raise PartialRoundPublicCypherKey to the power of own Private Cypher Key
	g.Exp(in.PartialRoundPublicCypherKey, keys.Z, out.PartialRoundPublicCypherKey)
	return out

}
