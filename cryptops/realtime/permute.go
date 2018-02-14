// Implements the Realtime Permute phase

package realtime

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
)

// The realtime Permute phase blindly permutes, then re-encrypts messages
// and recipient IDs to prevent other nodes or outside actors from knowing
// the origin of the messages.
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
	S *cyclic.Int
	V *cyclic.Int
}

// Pre-allocate memory and arrange key objects for Realtime Permute phase
func (p Permute) Build(g *cyclic.Group, face interface{}) *services.DispatchBuilder {
	// The empty interface should be castable to a Round
	round := face.(*globals.Round)

	// Allocate messages for output
	om := make([]services.Slot, round.BatchSize)

	// BEGIN CRYPTOGRAPHIC PORTION OF BUILD
	buildCryptoPermute(round, om)
	// END CRYPTOGRAPHIC PORTION OF BUILD

	keys := make([]services.NodeKeys, round.BatchSize)

	// Prepare the correct keys
	for i := uint64(0); i < round.BatchSize; i++ {
		keySlc := &KeysPermute{S: round.S[i], V: round.V[i]}

		keys[i] = keySlc
	}

	db := services.DispatchBuilder{BatchSize: round.BatchSize, Keys: &keys,
		Output: &om, G: g}

	return &db
}

// Input: Encrypted message, from Decrypt Phase
//        Encrypted recipient ID, from Decrypt Phase
// This phase permutes the message and the recipient ID and encrypts
// them with their respective permuted internode keys.
func (p Permute) Run(g *cyclic.Group, in, out *SlotPermute,
	keys *KeysPermute) services.Slot {

	// Eq 4.10 Multiply the message by its permuted key to make the permutation
	// secret to the previous node
	g.Mul(in.EncryptedMessage, keys.S, out.EncryptedMessage)

	// Eq 4.12 Multiply the recipient ID by its permuted key making the permutation
	// secret to the previous node
	g.Mul(in.EncryptedRecipientID, keys.V, out.EncryptedRecipientID)

	return out
}

func buildCryptoPermute(round *globals.Round, outMessages []services.Slot) {
	// Prepare the permuted output messages
	for i := uint64(0); i < round.BatchSize; i++ {
		slot := &SlotPermute{
			Slot:                 round.Permutations[i],
			EncryptedMessage:     cyclic.NewMaxInt(),
			EncryptedRecipientID: cyclic.NewMaxInt(),
		}
		// If this is the last node, save the EncryptedMessage
		if round.LastNode.EncryptedMessage != nil {
			slot.EncryptedMessage = round.LastNode.EncryptedMessage[i]
		}
		outMessages[i] = slot
	}
}
