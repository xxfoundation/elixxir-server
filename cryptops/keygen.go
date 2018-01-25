package cryptops

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/node"
	"gitlab.com/privategrity/server/services"
)

type GenerateClientKey struct{}

type KeysGenerateClientKey struct {
	sharedKeyStorage []byte
}

func (g GenerateClientKey) Run(group *cyclic.Group, in, out *KeySlot, keys *KeysGenerateClientKey) *KeySlot {
	return out
}

func (g GenerateClientKey) Build(group *cyclic.Group, face interface{}) *services.DispatchBuilder {

	// Get round from the empty interface
	round := face.(*node.Round)

	// Let's have 65536-bit long keys for now. We can increase or reduce
	// size as needed after profiling
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
