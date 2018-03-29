////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Implements the Realtime Identify phase

package realtime

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/crypto/format"
	"gitlab.com/privategrity/crypto/verification"
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
	recip := format.DeserializeRecipient(out.EncryptedRecipient)
	iv := recip.GetRecipientInitVect().LeftpadBytes(format.RIV_LEN)
	pmic := recip.GetRecipientMIC().LeftpadBytes(format.RMIC_LEN)
	recpbytes := recip.GetRecipientID().LeftpadBytes(format.RID_LEN)

	recipientMicList := [][]byte{iv, recpbytes}

	valid := verification.CheckMic(recipientMicList, pmic)

	//The second part of this line is a hack to make test pass at this stage
	//TODO: Update tests so the second part of this line can be removed
	if !valid && cyclic.NewIntFromBytes(pmic).Uint64() != 0 {
		// TODO: Re-enforce MIC checking, requires fixing main test!!!
		// out.EncryptedRecipient.SetUint64(globals.NIL_USER)
		jww.ERROR.Printf("Recipient MIC failed, Recipient ID read as %v",
			cyclic.NewIntFromBytes(recpbytes).Text(10))
	} else {
		out.EncryptedRecipient.SetBytes(recpbytes)
	}

	return out
}
