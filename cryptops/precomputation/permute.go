// Implements the Precomputation Permute phase

package precomputation

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/node"
	"gitlab.com/privategrity/server/services"
)

// Permute phase TODO
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
	// TODO
}

// Allocated memory and arranges key objects for the Precomputation Permute Phase
func (p Permute) Build(g *cyclic.Group, face interface{}) *services.DispatchBuilder {

	// Get round from the empty interface
	round := face.(*node.Round)

	// Allocate Memory for output
	om := make([]services.Slot, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = &SlotPermute{slot: i,
			EncryptedMessageKeys:         cyclic.NewMaxInt(),
			EncryptedRecipientIDKeys:     cyclic.NewMaxInt(),
			PartialMessageCypherText:     cyclic.NewMaxInt(),
			PartialRecipientIDCypherText: cyclic.NewMaxInt()}
	}

	keys := make([]services.NodeKeys, round.BatchSize)

	// Link the keys for permutation
	for i := uint64(0); i < round.BatchSize; i++ {
		keySlc := &KeysPermute{} // TODO
		keys[i] = keySlc
	}

	db := services.DispatchBuilder{BatchSize: round.BatchSize, Keys: &keys, Output: &om, G: g}

	return &db

}

func (p Permute) Run(g *cyclic.Group, in, out *SlotPermute, keys *KeysPermute) services.Slot {

	// Create Temporary variable
	// tmp := cyclic.NewMaxInt()

	// TODO

	return out

}
