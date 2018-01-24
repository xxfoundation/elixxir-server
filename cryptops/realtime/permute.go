// Implements the Realtime Permute phase

package realtime

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/node"
	"gitlab.com/privategrity/server/services"
)

// Permute implements the Permute phase of the precomputation.
type Permute struct{}

// (p Permute) Run() uses SlotPermute structs to pass data into and
// out of Permute
type SlotPermute struct {
	// Slot number
	Slot uint64

	// Encrypted message (permuted to a different slot in Run())
	EncryptedMessage *cyclic.Int
	// Encrypted recipient ID (permuted to a different slot in Run())
	EncryptedRecipientID *cyclic.Int
}

// SlotID() gets the slot number
func (p *SlotPermute) SlotID() uint64 {
	return p.Slot
}

// KeysPermute holds the keys used by the Permute operation
type KeysPermute struct {
	PermutedInternodeMessageKey   *cyclic.Int
	PermutedInternodeRecipientKey *cyclic.Int
}

// Pre-allocate memory and arrange key objects for Precomputation Permute phase
func (p Permute) Build(g *cyclic.Group, face interface{}) *services.DispatchBuilder {
	// The empty interface should be castable to a Round
	round := face.(*node.Round)

	// Allocate messages for output
	om := make([]services.Slot, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		// This is where the mixing happens: setting the slot to the
		// permutations that have been precomputed for this round
		// will put the message and recipient ID in a different slot
		// when we Run the encryption.
		om[i] = &SlotPermute{
			Slot:                 round.Permutations[i],
			EncryptedMessage:     cyclic.NewMaxInt(),
			EncryptedRecipientID: cyclic.NewMaxInt(),
		}
	}

	keys := make([]services.NodeKeys, round.BatchSize)

	// Prepare the correct keys
	for i := uint64(0); i < round.BatchSize; i++ {
		keySlc := &KeysPermute{PermutedInternodeMessageKey: round.S[i],
			PermutedInternodeRecipientKey: round.V[i]}

		keys[i] = keySlc
	}

	db := services.DispatchBuilder{BatchSize: round.BatchSize, Keys: &keys,
		Output: &om, G: g}

	return &db
}

func (p Permute) Run(g *cyclic.Group, in, out *SlotPermute,
	keys *KeysPermute) services.Slot {
	// Input: Encrypted message, from Decrypt Phase
	//        Encrypted recipient ID, from Decrypt Phase
	// This phase permutes the message and the recipient ID and encrypts
	// them with their respective permuted internode keys.

	// Multiply the message by its permuted key to make the permutation
	// secret to the previous node
	g.Mul(in.EncryptedMessage, keys.PermutedInternodeMessageKey,
		out.EncryptedMessage)

	// Multiply the recipient ID by its permuted key making the permutation
	// secret to the previous node
	g.Mul(in.EncryptedRecipientID, keys.PermutedInternodeRecipientKey,
		out.EncryptedRecipientID)

	return out
}
