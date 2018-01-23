// Implements the Realtime Encrypt phase

package realtime

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/node"
	"gitlab.com/privategrity/server/services"
)

// Encrypt phase TODO
type Encrypt struct{}

// SlotEncrypt is used to pass external data into Encrypt and to pass the results out of Encrypt
type SlotEncrypt struct {
	//Slot Number of the Data
	slot uint64
	// TODO
}

// SlotID Returns the Slot number
func (e *SlotEncrypt) SlotID() uint64 {
	return e.slot
}

// KeysEncrypt holds the keys used by the Encrypt Operation
type KeysEncrypt struct {
	// TODO
}

// Allocated memory and arranges key objects for the Realtime Encrypt Phase
func (e Encrypt) Build(g *cyclic.Group, face interface{}) *services.DispatchBuilder {

	// Get round from the empty interface
	round := face.(*node.Round)

	// Allocate Memory for output
	om := make([]services.Slot, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = &SlotEncrypt{
			slot: i,
		} // TODO
	}

	keys := make([]services.NodeKeys, round.BatchSize)

	// Link the keys for encryption
	for i := uint64(0); i < round.BatchSize; i++ {
		keySlc := &KeysEncrypt{} // TODO
		keys[i] = keySlc
	}

	db := services.DispatchBuilder{BatchSize: round.BatchSize, Keys: &keys, Output: &om, G: g}

	return &db

}

// TODO
func (e Encrypt) Run(g *cyclic.Group, in, out *SlotEncrypt, keys *KeysEncrypt) services.Slot {

	// Create Temporary variable
	// tmp := cyclic.NewMaxInt()

	// TODO

	return out

}
