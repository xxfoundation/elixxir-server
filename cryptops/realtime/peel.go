// Implements the Realtime Peel phase

package realtime

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/node"
	"gitlab.com/privategrity/server/services"
)

// Peel phase TODO
type Peel struct{}

// SlotPeel is used to pass external data into Peel and to pass the results out of Peel
type SlotPeel struct {
	//Slot Number of the Data
	slot uint64
	// TODO
}

// SlotID Returns the Slot number
func (e *SlotPeel) SlotID() uint64 {
	return e.slot
}

// KeysPeel holds the keys used by the Peel Operation
type KeysPeel struct {
	// TODO
}

// Allocated memory and arranges key objects for the Realtime Peel Phase
func (p Peel) Build(g *cyclic.Group, face interface{}) *services.DispatchBuilder {

	// Get round from the empty interface
	round := face.(*node.Round)

	// Allocate Memory for output
	om := make([]services.Slot, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = &SlotPeel{
			slot: i,
		} // TODO
	}

	keys := make([]services.NodeKeys, round.BatchSize)

	// Link the keys for peeling
	for i := uint64(0); i < round.BatchSize; i++ {
		keySlc := &KeysPeel{} // TODO
		keys[i] = keySlc
	}

	db := services.DispatchBuilder{BatchSize: round.BatchSize, Keys: &keys, Output: &om, G: g}

	return &db

}

// TODO
func (p Peel) Run(g *cyclic.Group, in, out *SlotPeel, keys *KeysPeel) services.Slot {

	// Create Temporary variable
	// tmp := cyclic.NewMaxInt()

	// TODO

	return out

}
