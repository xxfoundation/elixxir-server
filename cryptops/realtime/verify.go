////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Implements the Realtime Identify phase

package realtime

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/verification"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/services"
)

// Identify implements the Verification of the MIC in realtime processing.
// It checks the MIC and then .
type Verify struct{}

// KeysVerify holds the location to store the result of the MIC
type KeysVerify struct {
	// pointer to the location to store if the mic worked
	Verification *bool
}

// Pre-allocate memory and arrange key objects for realtime Verify
func (i Verify) Build(g *cyclic.Group,
	face interface{}) *services.DispatchBuilder {

	// The empty interface should be castable to a Round
	round := face.(*globals.Round)

	// Allocate messages for output
	om := make([]services.Slot, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = &Slot{Slot: i,
			EncryptedRecipient: cyclic.NewMaxInt(),
		}
	}

	keys := make([]services.NodeKeys, round.BatchSize)

	// Prepare the correct keys
	for i := uint64(0); i < round.BatchSize; i++ {
		keySlc := &KeysVerify{
			Verification: &round.MIC_Verification[i]}
		keys[i] = keySlc
	}

	db := services.DispatchBuilder{
		BatchSize: round.BatchSize,
		Keys:      &keys,
		Output:    &om, G: g}

	return &db
}

// Input: Decrypted Recipient ID Payload, from Identify phase
// This verifies the decrypted payload matches its MIC
func (i Verify) Run(g *cyclic.Group,
	in, out *Slot, keys *KeysVerify) services.Slot {

	recip := format.DeserializeRecipient(in.EncryptedRecipient.LeftpadBytes(format.TOTAL_LEN))
	iv := recip.GetRecipientInitVect()
	pmic := recip.GetRecipientMIC()
	recpbytes := recip.GetRecipientID()

	recipientMicList := [][]byte{iv, recpbytes}

	valid := verification.CheckMic(recipientMicList, pmic)

	if !valid {
		jww.WARN.Printf("Recipient MIC failed, Recipient ID read as %v",
			cyclic.NewIntFromBytes(recpbytes).Text(10))
		*keys.Verification = false
	} else {
		*keys.Verification = true
	}

	out.EncryptedRecipient.SetBytes(recpbytes)

	return out
}
