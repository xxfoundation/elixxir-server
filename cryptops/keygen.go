// Implements client key generation
package cryptops

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/crypto/forward"
	"gitlab.com/privategrity/server/node"
	"gitlab.com/privategrity/server/services"
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
}

// Dummy struct for runtime polymorphism requirements
type SlotGenerateClientKeyOut struct{}

func (s SlotGenerateClientKeyOut) SlotID() uint64 { return uint64(0) }

func (s SlotGenerateClientKeyOut) UserID() uint64 { return uint64(0) }

func (s SlotGenerateClientKeyOut) Key() *cyclic.Int { return nil }

func (s SlotGenerateClientKeyOut) GetKeyType() KeyType {
	var result KeyType
	return result
}

// Build() pre-allocates the memory and structs required to Run() this cryptop.
// This includes
// To correctly run this cryptop, you also need to prepare the user registry.
func (g GenerateClientKey) Build(group *cyclic.Group,
	face interface{}) *services.DispatchBuilder {

	// Get round from the empty interface
	round := face.(*node.Round)

	// Let's have 65536-bit long keys for now. We can increase or reduce
	// size as needed after profiling, or perhaps look for a way to reuse
	// these buffers more aggressively.
	keys := make([]services.NodeKeys, round.BatchSize)
	for i := uint64(0); i < round.BatchSize; i++ {
		keySlc := &KeysGenerateClientKey{
			sharedKeyStorage: make([]byte, 0, 8192),
		}
		keys[i] = keySlc
	}

	// outputMessages isn't really used for anything, but because of
	// dispatcher implementation details we still need to allocate
	// a few empty structs
	om := make([]services.Slot, round.BatchSize)
	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = SlotGenerateClientKeyOut{}
	}

	return &services.DispatchBuilder{round.BatchSize, &keys, &om, group}
}

// Run() generates a client key (either transmission or reception) through
// the dispatcher. The transmission key is used in the realtime Decrypt phase
// when the first node receives the message from the client, and the reception
// key is used after the realtime Peel phase, when the client is receiving the
// message from the last node.
func (g GenerateClientKey) Run(group *cyclic.Group, in, out KeySlot,
	keys *KeysGenerateClientKey) services.Slot {
	// This cryptop gets user information from the user registry, which is
	// an approach that isolates data less than I'd like.
	user := node.Users.GetUser(in.UserID())

	// Running this puts the next recursive key in the user's record and
	// the correct shared key for the key type into `in`'s key. Unlike
	// other cryptops, nothing goes in `out`: it's all mutated in place.
	if in.GetKeyType() == TRANSMISSION {
		forward.GenerateSharedKey(group, user.Transmission.BaseKey,
			user.Transmission.RecursiveKey, in.Key(),
			keys.sharedKeyStorage)
	} else if in.GetKeyType() == RECEPTION {
		forward.GenerateSharedKey(group, user.Reception.BaseKey,
			user.Reception.RecursiveKey, in.Key(),
			keys.sharedKeyStorage)
	}

	return in.(services.Slot)
}
