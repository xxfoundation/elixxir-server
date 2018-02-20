// Copyright © 2018 Privategrity Corporation
//
// All rights reserved.

package precomputation

import (
	"gitlab.com/privategrity/crypto/cyclic"
)

// These are all the dispatch slot types for precomputation dispatcher
// implementations.

// SlotGeneration is empty; no data being passed in or out of Generation
type SlotGeneration struct {
	//Slot Number of the Data
	Slot uint64
}

// SlotShare is used to pass external data into Share and to pass the results out of Share
type SlotShare struct {
	// Slot Number of the Data
	Slot uint64
	// Eq 10.3: Partial result of raising the global generator to the power of each node's Private Cypher Key
	PartialRoundPublicCypherKey *cyclic.Int
}

// SlotDecrypt is used to pass external data into Decrypt and to pass the results out of Decrypt
type SlotDecrypt struct {
	//Slot Number of the Data
	Slot uint64
	// Eq 12.9: First unpermuted internode message key from previous node
	EncryptedMessageKeys *cyclic.Int
	// Eq 12.11: First unpermuted internode recipient ID key from previous node
	EncryptedRecipientIDKeys *cyclic.Int
	// Eq 12.13: Partial cipher test for first unpermuted internode message key from previous node
	PartialMessageCypherText *cyclic.Int
	// Eq 12.15: Partial cipher test for first unpermuted internode recipient ID key from previous node
	PartialRecipientIDCypherText *cyclic.Int
}

// SlotPermute is used to pass external data into Permute and to pass the
// results out of Permute
type SlotPermute struct {
	//Slot Number of the Data
	Slot uint64
	// All of the first unpermuted internode message keys multiplied
	// together under Homomorphic Encryption
	EncryptedMessageKeys *cyclic.Int
	// All of the unpermuted internode recipient keys multiplied together
	// under Homomorphic Encryption
	EncryptedRecipientIDKeys *cyclic.Int
	// Partial Cypher Text for EncryptedMessageKeys
	PartialMessageCypherText *cyclic.Int
	// Partial Cypher Text for RecipientIDKeys
	PartialRecipientIDCypherText *cyclic.Int
}

// SlotEncrypt is used to pass external data into Encrypt and to pass the results out of Encrypt
type SlotEncrypt struct {
	// Slot Number of the Data
	Slot uint64
	// Partial Precomputation for the Messages
	EncryptedMessageKeys *cyclic.Int
	// Partial Cypher Text for the Message Precomputation
	PartialMessageCypherText *cyclic.Int
}

// (r Reveal) Run() uses SlotReveal structs to pass data into and out of Reveal
type SlotReveal struct {
	// Slot number
	Slot uint64

	// Partially decrypted message cypher texts
	PartialMessageCypherText *cyclic.Int
	// Partially decrypted recipient cypher texts
	PartialRecipientCypherText *cyclic.Int
}

// SlotStripIn is used to pass external data into Strip
type SlotStripIn struct {
	//Slot Number of the Data
	Slot uint64
	// Encrypted but completed message precomputation
	RoundMessagePrivateKey *cyclic.Int
	// Encrypted but completed recipient precomputation
	RoundRecipientPrivateKey *cyclic.Int
}

// SlotStripOut is used to pass the results out of Strip
type SlotStripOut struct {
	//Slot Number of the Data
	Slot uint64
	// Completed Message Precomputation
	MessagePrecomputation *cyclic.Int
	// Completed Recipient Precomputation
	RecipientPrecomputation *cyclic.Int
}

// SlotID functions for the above

// SlotID Returns the Slot number
func (e *SlotGeneration) SlotID() uint64 {
	return e.Slot
}

// SlotID Returns the Slot number
func (e SlotShare) SlotID() uint64 {
	return e.Slot
}

// SlotID Returns the Slot number of the input
func (e *SlotStripIn) SlotID() uint64 {
	return e.Slot
}

// SlotID Returns the Slot number of the output
func (e *SlotStripOut) SlotID() uint64 {
	return e.Slot
}

// SlotID Returns the Slot number
func (e *SlotDecrypt) SlotID() uint64 {
	return e.Slot
}

// SlotID Returns the Slot number
func (e *SlotPermute) SlotID() uint64 {
	return e.Slot
}

// SlotID Returns the Slot number
func (e SlotEncrypt) SlotID() uint64 {
	return e.Slot
}

// SlotID() gets the Slot number
func (r *SlotReveal) SlotID() uint64 {
	return r.Slot
}
