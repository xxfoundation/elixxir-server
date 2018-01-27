// Implements the Realtime Identify phase

package realtime

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/node"
	"gitlab.com/privategrity/server/services"
)

// Identify implements the Identify phase of the realtime processing.
// It removes the keys U and V that encrypt the recipient ID, so that we can
// start sending the ciphertext to the correct recipient.
type Identify struct{}

// (i Identify) Run() uses SlotIdentify structs to pass data into and out of
// Identify.
type SlotIdentify struct {
	// Slot number
	Slot uint64

	// It is encrypted until this phase has run on all the nodes
	EncryptedRecipientID *cyclic.Int
}

// SlotID() gets the Slot number
func (i *SlotIdentify) SlotID() uint64 {
	return i.Slot
}

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
	round := face.(*node.Round)

	// Allocate messages for output
	om := make([]services.Slot, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = &SlotIdentify{Slot: i,
			EncryptedRecipientID: cyclic.NewMaxInt(),
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
func (i Identify) Run(g *cyclic.Group, in, out *SlotIdentify,
	keys *KeysIdentify) services.Slot {

	// Eq 5.1
	// Multiply EncryptedRecipientID by the precomputed value
	g.Mul(in.EncryptedRecipientID, keys.RecipientPrecomputation,
		out.EncryptedRecipientID)

	return out
}
