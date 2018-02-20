// Copyright © 2018 Privategrity Corporation
//
// All rights reserved.
//
// These are the dispatch slot types for precomputation dispatcher
// implementations. There are 3 types: Generate, Share, and Precomputation.

package precomputation

import (
	"gitlab.com/privategrity/crypto/cyclic"
)

// SlotGeneration is empty; no data being passed in or out of Generation.
// The Generate dispatcher is a control signal to tell each node to
// generate their keys for each message slot.
type SlotGeneration struct {
	Slot uint64
}

// SlotShare is used to pass a round-wide Public Cypher Key to each node
type SlotShare struct {
	Slot uint64
	// Eq 10.3: Partial result of raising the global generator to the
	//          power of each node's Private Cypher Key
	PartialRoundPublicCypherKey *cyclic.Int
}

// PrecomputationSlot is a general slot structure used by all other
// precomputation cryptops. The semantics of each element change and not
// all elements are used by every cryptop, but the purpose remains the same
// as the data travels through precomputation.
type PrecomputationSlot struct {
	Slot uint64
	// Starts as message partial cypher text and becomes message precomputation
	MessagePrecomputation *cyclic.Int
	// Encrypted message key to round message private key
	MessageCypher *cyclic.Int
	// Receiptient ID partial cypher text, becomes receipient id precomputation
	RecipientIDPrecomputation *cyclic.Int
	// Encrypted recipient id key to round recipient id private key
	RecipientIDCypher *cyclic.Int
}

// SlotID functions return the Slot number
func (e *SlotGeneration) SlotID() uint64 {
	return e.Slot
}

func (e SlotShare) SlotID() uint64 {
	return e.Slot
}

func (e *PrecomputationSlot) SlotID() uint64 {
	return e.Slot
}
