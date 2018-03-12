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
	"gitlab.com/privategrity/crypto/verification"
	jww "github.com/spf13/jwalterweatherman"
	"fmt"
)

const (
	TOTAL_LEN 		uint64 = 512

	// Length and Position of the Initialization Vector for both the payload and
	// the recipient
	IV_LEN			uint64 = 9
	IV_START		uint64 = 0
	IV_END			uint64 = IV_LEN

	// Length and Position of message payload
	PAYLOAD_LEN   	uint64 = TOTAL_LEN-SID_LEN-IV_LEN-PMIC_LEN
	PAYLOAD_START	uint64 = IV_END
	PAYLOAD_END		uint64 = PAYLOAD_START+PAYLOAD_LEN

	// Length and Position of the Sender ID in the payload
	SID_LEN   		uint64 = 8
	SID_START		uint64 = PAYLOAD_END
	SID_END			uint64 = SID_START + SID_LEN

	// Length and Position of the Payload MIC
	PMIC_LEN	    uint64 = 8
	PMIC_START		uint64 = SID_END
	PMIC_END		uint64 = PMIC_START+PMIC_LEN

	// Length and Position of the Recipient ID
	RID_LEN 		uint64 = TOTAL_LEN-IV_LEN-RMIC_LEN
	RID_START		uint64 = IV_END
	RID_END			uint64 = RID_START+RID_LEN

	// Length and Position of the Recipient MIC
	RMIC_LEN	    uint64 = 8
	RMIC_START		uint64 = RID_END
	RMIC_END		uint64 = RMIC_START+RMIC_LEN
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
	recpbytes := out.EncryptedRecipient.LeftpadBytes(TOTAL_LEN)
	iv := recpbytes[IV_START:IV_END]
	pmic := recpbytes[RMIC_START:RMIC_END]
	recpbytes = recpbytes[RID_START:RID_END]

	recipientMicList := [][]byte{ iv,recpbytes}

	valid := verification.CheckMic(recipientMicList,pmic)

	if !valid{
		out.EncryptedRecipient.SetUint64(globals.
			NIL_USER)
		jww.ERROR.Printf("Recipient MIC failed, Recipient ID read as %v, " +
			"defaulting to nil user\n", cyclic.NewIntFromBytes(recpbytes).Text(
				10))
	}else{
		out.EncryptedRecipient.SetBytes(recpbytes)
	}

	return out
}
