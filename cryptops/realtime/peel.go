// Implements the Realtime Peel phase

package realtime

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/node"
	"gitlab.com/privategrity/server/services"
)

// Peel phase removes the Internode Keys by multiplying in the precomputation to the encrypted message
type Peel struct{}

// SlotPeel is used to pass external data into Peel and to pass the results out of Peel
type SlotPeel struct {
	//Slot Number of the Data
	slot uint64
	//ID of the client who will recieve the message (Pass through)
	RecipientID uint64
	// Permuted Message encrypted by all internode keys and the reception keys
	EncryptedMessage *cyclic.Int
}

// SlotID Returns the Slot number
func (e *SlotPeel) SlotID() uint64 {
	return e.slot
}

// KeysPeel holds the keys used by the Peel Operation
type KeysPeel struct {
	// All message internode keys multiplied together
	MessagePrecomputation *cyclic.Int
}

// Allocated memory and arranges key objects for the Realtime Peel Phase
func (p Peel) Build(g *cyclic.Group, face interface{}) *services.DispatchBuilder {

	// Get round from the empty interface
	round := face.(*node.Round)

	// Allocate Memory for output
	om := make([]services.Slot, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = &SlotPeel{
			slot:             i,
			EncryptedMessage: cyclic.NewMaxInt(),
			RecipientID:      0,
		}
	}

	keys := make([]services.NodeKeys, round.BatchSize)

	// Link the keys for peeling
	for i := uint64(0); i < round.BatchSize; i++ {
		keySlc := &KeysPeel{
			MessagePrecomputation: round.LastNode.MessagePrecomputation[i],
		}
		keys[i] = keySlc
	}

	db := services.DispatchBuilder{BatchSize: round.BatchSize, Keys: &keys, Output: &om, G: g}

	return &db

}

// Removes the Internode Keys by multiplying in the precomputation to the encrypted message
func (p Peel) Run(g *cyclic.Group, in, out *SlotPeel, keys *KeysPeel) services.Slot {

	// Eq 7.1: Multiply in the precomputation
	g.Mul(in.EncryptedMessage, keys.MessagePrecomputation, out.EncryptedMessage)

	// Pass through SenderID
	out.RecipientID = in.RecipientID

	return out

}
