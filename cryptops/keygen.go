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

func (g GenerateClientKey) Run(group *cyclic.Group, in, out KeySlot,
	keys *KeysGenerateClientKey) KeySlot {
	user := node.GetUser(in.UserID())

	if in.GetKeyType() == TRANSMISSION {
		forward.GenerateSharedKey(group, user.Transmission.BaseKey,
			user.Transmission.RecursiveKey, out.Key(),
			keys.sharedKeyStorage)
	} else if in.GetKeyType() == RECEPTION {
		forward.GenerateSharedKey(group, user.Reception.BaseKey,
			user.Reception.RecursiveKey, out.Key(),
			keys.sharedKeyStorage)
	}

	return out
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

	// outputMessages is nil because we store data to the user instead
	return &services.DispatchBuilder{round.BatchSize, &keys, nil, group}
}
