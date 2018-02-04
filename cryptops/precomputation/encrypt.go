package precomputation

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
)

// Encrypt implements the Encrypt Phase of the Precomputation. It adds the
// Seconds Unpermuted Internode Message Keys into the Message Precomputation
type Encrypt struct{}

// SlotEncrypt is used to pass external data into Encrypt and to pass the results out of Encrypt
type SlotEncrypt struct {
	// Slot Number of the Data
	Slot uint64
	// Partial Precomputation for the Messages
	EncryptedMessageKeys *cyclic.Int
	// Partial Cypher Text for the Message Precomputation
	PartialMessageCypherText *cyclic.Int
}

// SlotID Returns the Slot number
func (e SlotEncrypt) SlotID() uint64 {
	return e.Slot
}

// KeysEncrypt holds the keys used by the Encrypt Operation
type KeysEncrypt struct {
	// Public Key for entire round generated in Share Phase
	CypherPublicKey *cyclic.Int
	// Inverse Second Unpermuted Internode Message Key
	T_INV *cyclic.Int
	// Second Unpermuted Internode Message Private Key
	Y_T *cyclic.Int
}

// Allocated memory and arranges key objects for the Precomputation Encrypt Phase
func (e Encrypt) Build(g *cyclic.Group, face interface{}) *services.DispatchBuilder {

	// Get round from the empty interface
	round := face.(*globals.Round)

	// Allocate Memory for output
	om := make([]services.Slot, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = &SlotEncrypt{
			Slot:                     i,
			EncryptedMessageKeys:     cyclic.NewMaxInt(),
			PartialMessageCypherText: cyclic.NewMaxInt(),
		}
	}

	keys := make([]services.NodeKeys, round.BatchSize)

	// Link the keys for encryption
	for i := uint64(0); i < round.BatchSize; i++ {
		keySlc := &KeysEncrypt{
			CypherPublicKey: round.CypherPublicKey,
			T_INV:           round.T_INV[i],
			Y_T:             round.Y_T[i],
		}
		keys[i] = keySlc
	}

	db := services.DispatchBuilder{BatchSize: round.BatchSize, Keys: &keys, Output: &om, G: g}

	return &db

}

// Output of the Permute Phase is passed to the first Node which multiplies in its Encrypted
// Second Unpermuted Message Keys and the associated Private Keys into the Partial Message Cypher Test.
func (e Encrypt) Run(g *cyclic.Group, in, out *SlotEncrypt, keys *KeysEncrypt) services.Slot {

	// Create Temporary variable
	tmp := cyclic.NewMaxInt()

	// Eq 11.1: Homomorphicly encrypt the Inverse Second Unpermuted Internode Message Key.
	g.Exp(g.G, keys.Y_T, tmp)
	g.Mul(keys.T_INV, tmp, tmp)

	// Eq 14.5: Multiply the encrypted Inverse Second Unpermuted Internode Message Key into the partial precomputation.
	g.Mul(in.EncryptedMessageKeys, tmp, out.EncryptedMessageKeys)

	// Eq 11.2: Create the Inverse Second Unpermuted Internode Message Public Key.
	g.Exp(keys.CypherPublicKey, keys.Y_T, tmp)

	// Multiply cypher the Inverse Second Unpermuted Internode Message Public Key into the Partial Cypher Text.
	g.Mul(in.PartialMessageCypherText, tmp, out.PartialMessageCypherText)

	return out

}
