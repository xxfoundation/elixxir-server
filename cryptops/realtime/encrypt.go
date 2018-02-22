////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Implements the Realtime Encrypt phase

package realtime

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
)

// The Encrypt phase adds in the final internode keys while simultaneously
// adding in Reception Keys for the recipient.
type Encrypt struct{}

// KeysEncrypt holds the keys used by the Encrypt Operation
type KeysEncrypt struct {
	// Eq 6.6: Second Unpermuted Internode Message Key
	T *cyclic.Int
}

// Allocated memory and arranges key objects for the Realtime Encrypt Phase
func (e Encrypt) Build(g *cyclic.Group,
	face interface{}) *services.DispatchBuilder {

	// Get round from the empty interface
	round := face.(*globals.Round)

	// Allocate Memory for output
	om := make([]services.Slot, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = &RealtimeSlot{
			Slot:      i,
			Message:   cyclic.NewMaxInt(),
			CurrentID: 0,
		}
	}

	keys := make([]services.NodeKeys, round.BatchSize)

	// Link the keys for encryption
	for i := uint64(0); i < round.BatchSize; i++ {
		keySlc := &KeysEncrypt{
			T: round.T[i],
		}
		keys[i] = keySlc
	}

	db := services.DispatchBuilder{BatchSize: round.BatchSize,
		Keys: &keys, Output: &om, G: g}

	return &db

}

// Multiplies in the ReceptionKey and the node’s cypher key
func (e Encrypt) Run(g *cyclic.Group, in *RealtimeSlot,
	out *RealtimeSlot, keys *KeysEncrypt) services.Slot {

	// Create Temporary variable
	tmp := cyclic.NewMaxInt()

	// Eq 6.6: Multiplies the Reception Key and the Second Unpermuted
	// Internode Keys into the Encrypted Message
	g.Mul(in.CurrentKey, keys.T, tmp)
	g.Mul(in.Message, tmp, out.Message)

	// Pass through RecipientID
	out.CurrentID = in.CurrentID

	return out

}
