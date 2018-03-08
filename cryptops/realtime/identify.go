////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Implements the Realtime Identify phase

package realtime

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
)

// Identify implements the Identify phase of the realtime processing.
// It removes the keys U and V that encrypt the recipient ID, so that we can
// start sending the ciphertext to the correct recipient.
type Identify struct{}

// KeysIdentify holds the key needed for the realtime Identify phase
type KeysIdentify struct {
	// Result of the precomputation for the recipient ID
	// One of the two results of the precomputation
	RecipientPrecomputation *cyclic.Int
}

// Pre-allocate memory and arrange key objects for realtime Identify phase
func (i Identify) Build(g *cyclic.Group,
	face interface{}) *services.DispatchBuilder {

	// The empty interface should be castable to a Round
	round := face.(*globals.Round)

	// Allocate messages for output
	om := make([]services.Slot, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = &RealtimeSlot{Slot: i,
			EncryptedRecipient: cyclic.NewMaxInt(),
		}
	}

	keys := make([]services.NodeKeys, round.BatchSize)

	// Prepare the correct keys
	for i := uint64(0); i < round.BatchSize; i++ {
		keySlc := &KeysIdentify{
			RecipientPrecomputation: round.RecipientPrecomputation[i]}
		keys[i] = keySlc
	}

	db := services.DispatchBuilder{
		BatchSize: round.BatchSize,
		Keys:      &keys,
		Output:    &om, G: g}

	return &db
}

// Input: Encrypted Recipient ID, from Permute phase
// This phase decrypts the recipient ID, identifying the recipient
func (i Identify) Run(g *cyclic.Group,
	in, out *RealtimeSlot, keys *KeysIdentify) services.Slot {



	// Eq 5.1
	// Multiply EncryptedRecipientID by the precomputed value
	g.Mul(in.EncryptedRecipient, keys.RecipientPrecomputation,
		out.EncryptedRecipient)

	// These lines remove the nonce on the recipient ID,
	// so that the server can send the message to an untainted recipient
	recpbytes := out.EncryptedRecipient.LeftpadBytes(512)
	out.EncryptedRecipient.SetBytes(recpbytes[503:512])


	return out
}
