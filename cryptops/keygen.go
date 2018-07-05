////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Implements client key generation
package cryptops

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/privategrity/crypto/cyclic"
	cmix "gitlab.com/privategrity/crypto/messaging"
	"gitlab.com/privategrity/server/cryptops/realtime"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
)

//Denotes what kind of key will be
type KeyType uint8

const (
	TRANSMISSION KeyType = 1
	RECEPTION    KeyType = 2
	RECEIPT      KeyType = 3
	RETURN       KeyType = 4
)

// Generate client key creates shared keys for the client's transmission and
// reception and creates the next recursive key for that shared key using the
// current recursive key. These keys are used to encrypt and decrypt user
// messages at both ends of the Realtime phase.
type GenerateClientKey struct{}

// This byte slice should have lots of capacity to hold the long key for shared
// key generation
type KeysGenerateClientKey struct {
	sharedKeyStorage []byte
	keySelection     KeyType
	verification     *bool
}

// Build() pre-allocates the memory and structs required to Run() this cryptop.
// This includes
// To correctly run this cryptop, you also need to prepare the user registry.
func (g GenerateClientKey) Build(group *cyclic.Group,
	face interface{}) *services.DispatchBuilder {

	// Get round from the empty interface
	faceLst := face.([]interface{})
	round := faceLst[0].(*globals.Round)
	keySelection := faceLst[1].(KeyType)

	// Let's have 65536-bit long keys for now. We can increase or reduce
	// size as needed after profiling, or perhaps look for a way to reuse
	// these buffers more aggressively.
	keys := make([]services.NodeKeys, round.BatchSize)
	for i := uint64(0); i < round.BatchSize; i++ {
		keySlc := &KeysGenerateClientKey{
			make([]byte, 0, 8192),
			keySelection,
			&round.MIC_Verification[i],
		}
		keys[i] = keySlc
	}

	// outputMessages isn't really used for anything, but because of
	// dispatcher implementation details we still need to allocate
	// a few empty structs
	om := make([]services.Slot, round.BatchSize)
	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = &realtime.Slot{}
	}

	return &services.DispatchBuilder{round.BatchSize, &keys, &om, group}
}

// Run() generates a client key (either transmission or reception) through
// the dispatcher. The transmission key is used in the realtime Decrypt phase
// when the first node receives the message from the client, and the reception
// key is used after the realtime Peel phase, when the client is receiving the
// message from the last node.
func (g GenerateClientKey) Run(group *cyclic.Group, in,
	out *realtime.Slot,
	keys *KeysGenerateClientKey) services.Slot {
	// This cryptop gets user information from the user registry, which is
	// an approach that isolates data less than I'd like.

	user, err := globals.Users.GetUser(in.CurrentID)

	if err != nil {
		jww.INFO.Printf("GenerateClientKey Run: Got lookup"+
			" failure on %v, using a random key to destroy the data. Error: %s",
			in.CurrentID, err.Error())
		group.Random(in.CurrentKey)
		return in
	}

	// Running this puts the next recursive key in the user's record and
	// the correct shared key for the key type into `in`'s key. Unlike
	// other cryptops, nothing goes in `out`: it's all mutated in place.
	if keys.keySelection == TRANSMISSION {
		baseKey := user.Transmission.BaseKey
		decryptionKey := cmix.NewDecryptionKey(in.Salt, baseKey, group)
		in.CurrentKey.Set(decryptionKey)
	} else if keys.keySelection == RECEPTION {
		if !*keys.verification {
			jww.ERROR.Printf(
				"Key Generation Failed: MIC failure detected.\n"+
					"  Slot: %v; Received: %v", in.Slot, keys.keySelection)
		} else {
			baseKey := user.Reception.BaseKey
			encryptionKey := cmix.NewEncryptionKey(in.Salt, baseKey, group)
			in.CurrentKey.Set(encryptionKey)
		}

	} else {
		jww.FATAL.Panicf(
			"Key Generation Failed: Invalid Key Selection.\n"+
				"  Slot: %v; Received: %v", in.Slot, keys.keySelection)
	}

	globals.Users.UpsertUser(user)

	return in
}
