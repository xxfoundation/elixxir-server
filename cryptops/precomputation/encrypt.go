// Implements the Precomputation Encrypt phase

package precomputation

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/node"
	"gitlab.com/privategrity/server/services"
)

// Encrypt implements the Encrypt Phase of the Precomputation.  It adds the
// Seconds Unpermuted Internode Message Keys int the Message Precomputation
type Encrypt struct{}

// SlotEncrypt is used to pass external data into Encrypt and to
// pass the results out of Encrypt
type SlotEncrypt struct {
	//Slot Number of the Data
	slot uint64

	//Partial Precomputation for the Messages
	EncryptedMessageKeys *cyclic.Int
	//Partial Cypher Text for the Message Precomputation
	PartialMessageCypherText *cyclic.Int
}

// SlotID Returns the Slot number
func (e *SlotEncrypt) SlotID() uint64 {
	return e.slot
}

// KeysEncrypt holds the keys used by the Encrypt Operation
type KeysEncrypt struct {
	// Public Key for entire round generated in Share Phase
	PublicCypherKey *cyclic.Int
	// Inverse Second Unpermuted Internode Message Key
	T_INV *cyclic.Int
	// Second Unpermuted Internode Message Private Key
	Y_T *cyclic.Int
}

// Allocated memory and arranges key objects for the Precomputation Encrypt Phase
func (gen Encrypt) Build(g *cyclic.Group, face interface{}) *services.DispatchBuilder {

	//get round from the empty interface
	round := face.(*node.Round)

	//Allocate Memory for output
	om := make([]services.Slot, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = &SlotEncrypt{slot: i, EncryptedMessageKeys: cyclic.NewMaxInt(),
			PartialMessageCypherText: cyclic.NewMaxInt()}
	}

	keys := make([]services.NodeKeys, round.BatchSize)

	//Link the keys for encryption
	for i := uint64(0); i < round.BatchSize; i++ {
		keySlc := &KeysEncrypt{PublicCypherKey: round.CypherPublicKey,
			T_INV: round.T_INV[i], Y_T: round.Y_T[i]}
		keys = append(keys, keySlc)
	}

	db := services.DispatchBuilder{BatchSize: round.BatchSize, Keys: &keys, Output: &om, G: g}

	return &db

}

func (e Encrypt) Run(g *cyclic.Group, in, out *SlotEncrypt, keys *KeysEncrypt) services.Slot {
	// Output of the Permute Phase is passed to the first Node which multiplies in its Encrypted
	// Second Unpermuted Message Keys and the associated Private Keys into the Partial Message Cypher Test.

	// Create Temporary variable
	tmp := cyclic.NewMaxInt()

	// Homomorphicly encrypt the Inverse Second Unpermuted Internode Message Key. Eq 11.1
	g.Exp(g.G, keys.Y_T, tmp)
	g.Mul(keys.T_INV, tmp, tmp)

	// Multiply the encrypted Inverse Second Unpermuted Internode Message Key into
	// the partial precomputation. Eq 14.5
	g.Mul(in.EncryptedMessageKeys, tmp, out.EncryptedMessageKeys)

	// Create the Inverse Second Unpermuted Internode Message Public Key. Eq 11.2
	g.Exp(keys.PublicCypherKey, keys.Y_T, tmp)

	// Multiply cypher the Inverse Second Unpermuted Internode Message Public Key
	// into the Partial Cypher Text
	g.Mul(in.PartialMessageCypherText, tmp, out.PartialMessageCypherText)

	return out

}
