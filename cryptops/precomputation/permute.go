// Implements the Precomputation Permute phase

package precomputation

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/node"
	"gitlab.com/privategrity/server/services"
)

// Permute phase permutes the four results of the Decrypt phase and multiplies in own keys
type Permute struct{}

// SlotPermute is used to pass external data into Permute and to pass the results out of Permute
type SlotPermute struct {
	//Slot Number of the Data
	slot uint64
	// Eq 13.9
	EncryptedMessageKeys *cyclic.Int
	// Eq 13.11
	EncryptedRecipientIDKeys *cyclic.Int
	// Eq 13.13
	PartialMessageCypherText *cyclic.Int
	// Eq 13.15
	PartialRecipientIDCypherText *cyclic.Int
}

// SlotID Returns the Slot number
func (e *SlotPermute) SlotID() uint64 {
	return e.slot
}

// KeysPermute holds the keys used by the Permute Operation
type KeysPermute struct {
	// Public Key for entire round generated in Share Phase
	CypherPublicKey *cyclic.Int
	// Encrypted Inverse Permuted Internode Message Key
	S_INV *cyclic.Int
	// Encrypted Inverse Permuted Internode Recipient Key
	V_INV *cyclic.Int
	// Permuted Internode Message Partial Cypher Text
	Y_S *cyclic.Int
	// Permuted Internode Recipient Partial Cypher Text
	Y_V *cyclic.Int
}

// Allocated memory and arranges key objects for the Precomputation Permute Phase
func (p Permute) Build(g *cyclic.Group, face interface{}) *services.DispatchBuilder {

	// Get round from the empty interface
	round := face.(*node.Round)

	// Allocate Memory for output
	om := make([]services.Slot, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = &SlotPermute{
			slot:                         i,
			EncryptedMessageKeys:         cyclic.NewMaxInt(),
			EncryptedRecipientIDKeys:     cyclic.NewMaxInt(),
			PartialMessageCypherText:     cyclic.NewMaxInt(),
			PartialRecipientIDCypherText: cyclic.NewMaxInt(),
		}
	}

	keys := make([]services.NodeKeys, round.BatchSize)

	// Link the keys for permutation
	for i := uint64(0); i < round.BatchSize; i++ {
		keySlc := &KeysPermute{
			CypherPublicKey: round.CypherPublicKey,
			S_INV:           round.S_INV[i],
			V_INV:           round.V_INV[i],
			Y_S:             round.Y_S[i],
			Y_V:             round.Y_V[i],
		}
		keys[i] = keySlc
	}

	db := services.DispatchBuilder{BatchSize: round.BatchSize, Keys: &keys, Output: &om, G: g}

	return &db

}

// Permutes the four results of the Decrypt phase and multiplies in own keys
func (p Permute) Run(g *cyclic.Group, in, out *SlotPermute, keys *KeysPermute) services.Slot {

	// Create Temporary variable
	tmp := cyclic.NewMaxInt()

	// Eq 13.1
	g.Exp(g.G, keys.Y_S, tmp)
	g.Mul(keys.S_INV, tmp, tmp)
	g.Mul(in.EncryptedMessageKeys, tmp, out.EncryptedMessageKeys)

	// Eq 13.3
	g.Exp(g.G, keys.Y_V, tmp)
	g.Mul(keys.V_INV, tmp, tmp)
	g.Mul(in.EncryptedRecipientIDKeys, tmp, out.EncryptedRecipientIDKeys)

	// Eq 13.5
	g.Exp(keys.CypherPublicKey, keys.Y_S, tmp)
	g.Mul(in.PartialMessageCypherText, tmp, out.PartialMessageCypherText)

	// Eq 13.7
	g.Exp(keys.CypherPublicKey, keys.Y_V, tmp)
	g.Mul(in.PartialRecipientIDCypherText, tmp, out.PartialRecipientIDCypherText)

	return out

}
