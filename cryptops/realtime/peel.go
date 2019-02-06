////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Implements the Realtime Peel phase

package realtime

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/elixxir/primitives/userid"
)

// Peel phase removes the Internode Keys by multiplying in the precomputation
// to the encrypted message
type Peel struct{}

// KeysPeel holds the keys used by the Peel Operation
type KeysPeel struct {
	// All message internode keys multiplied together
	MessagePrecomputation *cyclic.Int
}

// Allocated memory and arranges key objects for the Realtime Peel Phase
func (p Peel) Build(g *cyclic.Group,
	face interface{}) *services.DispatchBuilder {

	// Get round from the empty interface
	round := face.(*globals.Round)

	// Allocate Memory for output
	om := make([]services.Slot, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = &Slot{
			Slot:      i,
			Message:   cyclic.NewMaxInt(),
			CurrentID: id.ZeroID,
		}
	}

	keys := make([]services.NodeKeys, round.BatchSize)

	// Link the keys for peeling
	for i := uint64(0); i < round.BatchSize; i++ {
		keySlc := &KeysPeel{
			MessagePrecomputation: round.LastNode.MessagePrecomputation[i],
		}
		keys[i] = keySlc
	}

	db := services.DispatchBuilder{BatchSize: round.BatchSize,
		Keys: &keys, Output: &om, G: g}

	return &db

}

// Removes the Internode Keys by multiplying
// in the precomputation to the encrypted message
func (p Peel) Run(g *cyclic.Group,
	in, out *Slot, keys *KeysPeel) services.Slot {

	// Eq 7.1: Multiply in the precomputation
	g.Mul(in.Message, keys.MessagePrecomputation, out.Message)

	// Pass through SenderID
	out.CurrentID = in.CurrentID
	out.Salt = in.Salt

	return out

}
