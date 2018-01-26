package cryptops

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/crypto/forward"
	"gitlab.com/privategrity/server/node"
	"gitlab.com/privategrity/server/services"
)

type GenerateClientKey struct{}

type KeysGenerateClientKey struct {
	sharedKeyStorage []byte
}

// dummy struct for runtime polymorphism requirements
type SlotGenerateClientKeyOut struct{}

func (s SlotGenerateClientKeyOut) SlotID() uint64 { return uint64(0) }

func (s SlotGenerateClientKeyOut) UserID() uint64 { return uint64(0) }

func (s SlotGenerateClientKeyOut) Key() *cyclic.Int { return nil }

func (s SlotGenerateClientKeyOut) GetKeyType() KeyType {
	var result KeyType
	return result
}

func (g GenerateClientKey) Run(group *cyclic.Group, in, out KeySlot,
	keys *KeysGenerateClientKey) services.Slot {
	user := node.GetUser(in.UserID())

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
