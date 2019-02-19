////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Implements the Realtime Encrypt phase

package realtime

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/services"
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
		om[i] = &Slot{
			Slot:       i,
			Message:    cyclic.NewMaxInt(),
			CurrentID:  id.ZeroID,
			CurrentKey: cyclic.NewMaxInt(),
			Salt:       make([]byte, 0),
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
func (e Encrypt) Run(g *cyclic.Group, in *Slot,
	out *Slot, keys *KeysEncrypt) services.Slot {

	encryptionKey := in.CurrentKey

	// Eq 6.6: Multiplies the Reception Key and the Second Unpermuted
	// Internode Keys into the Encrypted Message
	g.Mul(encryptionKey, in.Message, in.Message)
	g.Mul(keys.T, in.Message, out.Message)

	// Pass through RecipientID
	out.CurrentID = in.CurrentID
	out.Salt = in.Salt

	return out

}
