// Implements the Realtime Decrypt phase

package realtime

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/node"
	"gitlab.com/privategrity/server/services"
)

// Decrypt phase TODO
type Decrypt struct{}

// SlotDecrypt is used to pass external data into Decrypt and to pass the results out of Decrypt
type SlotDecrypt struct {
	//Slot Number of the Data
	slot uint64
	// TODO
}

// SlotID Returns the Slot number
func (e *SlotDecrypt) SlotID() uint64 {
	return e.slot
}

// KeysDecrypt holds the keys used by the Decrypt Operation
type KeysDecrypt struct {
	// TODO
}

// Allocated memory and arranges key objects for the Realtime Decrypt Phase
func (d Decrypt) Build(g *cyclic.Group, face interface{}) *services.DispatchBuilder {

	// Get round from the empty interface
	round := face.(*node.Round)

	// Allocate Memory for output
	om := make([]services.Slot, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = &SlotDecrypt{slot: i} // TODO
	}

	keys := make([]services.NodeKeys, round.BatchSize)

	// Link the keys for decryption
	for i := uint64(0); i < round.BatchSize; i++ {
		keySlc := &KeysDecrypt{} // TODO
		keys[i] = keySlc
	}

	db := services.DispatchBuilder{BatchSize: round.BatchSize, Keys: &keys, Output: &om, G: g}

	return &db

}

// TODO
func (d Decrypt) Run(g *cyclic.Group, in, out *SlotDecrypt, keys *KeysDecrypt) services.Slot {

	// Create Temporary variable
	// tmp := cyclic.NewMaxInt()

	// TODO

	return out

}
