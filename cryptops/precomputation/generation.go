// Implements the Precomputation Generation  phase

package precomputation

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/node"
	"gitlab.com/privategrity/server/services"
)

// Generation phase TODO
type Generation struct{}

// SlotGeneration is used to pass external data into Generation and to pass the results out of Generation
type SlotGeneration struct {
	//Slot Number of the Data
	slot uint64
	// TODO
}

// SlotID Returns the Slot number
func (e *SlotGeneration) SlotID() uint64 {
	return e.slot
}

// KeysGeneration holds the keys used by the Generation Operation
type KeysGeneration struct {
	// TODO
}

// Allocated memory and arranges key objects for the Precomputation Generation Phase
func (gen Generation) Build(g *cyclic.Group, face interface{}) *services.DispatchBuilder {

	// Get round from the empty interface
	round := face.(*node.Round)

	// Allocate Memory for output
	om := make([]services.Slot, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = &SlotGeneration{slot: i} // TODO
	}

	keys := make([]services.NodeKeys, round.BatchSize)

	// Link the keys for decryption
	for i := uint64(0); i < round.BatchSize; i++ {
		keySlc := &KeysGeneration{} // TODO
		keys[i] = keySlc
	}

	db := services.DispatchBuilder{BatchSize: round.BatchSize, Keys: &keys, Output: &om, G: g}

	return &db

}

func (gen Generation) Run(g *cyclic.Group, in, out *SlotGeneration, keys *KeysGeneration) services.Slot {

	// Create Temporary variable
	// tmp := cyclic.NewMaxInt()

	// TODO

	return out

}
