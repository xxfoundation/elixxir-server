// Implements the Realtime Encrypt phase

package realtime

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/node"
	"gitlab.com/privategrity/server/services"
)

// Encrypt phase generates a shared ReceptionKey between that node and the client
type Encrypt struct{}

// SlotEncrypt is used to pass external data into Encrypt
type SlotEncryptIn struct {
	//Slot Number of the Data
	slot uint64
	// Pass through
	RecipientID uint64
	// Eq 6.6
	EncryptedMessage *cyclic.Int
	// Eq 6.6
	ReceptionKey *cyclic.Int
}

// SlotEncryptOut is used to pass  the results out of Encrypt
type SlotEncryptOut struct {
	//Slot Number of the Data
	slot uint64
	// Pass through
	RecipientID uint64
	// Eq 6.7
	EncryptedMessage *cyclic.Int
}

// SlotID Returns the Slot number
func (e *SlotEncryptIn) SlotID() uint64 {
	return e.slot
}

// SlotID Returns the Slot number
func (e *SlotEncryptOut) SlotID() uint64 {
	return e.slot
}

// KeysEncrypt holds the keys used by the Encrypt Operation
type KeysEncrypt struct {
	// Eq 6.6: Second Unpermuted Internode Message Key
	T *cyclic.Int
}

// Allocated memory and arranges key objects for the Realtime Encrypt Phase
func (e Encrypt) Build(g *cyclic.Group, face interface{}) *services.DispatchBuilder {

	// Get round from the empty interface
	round := face.(*node.Round)

	// Allocate Memory for output
	om := make([]services.Slot, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = &SlotEncryptOut{
			slot:             i,
			EncryptedMessage: cyclic.NewMaxInt(),
			RecipientID:      0,
		}
	}

	keys := make([]services.NodeKeys, round.BatchSize)

	// Link the keys for encryption
	for i := uint64(0); i < round.BatchSize; i++ {
		keySlc := &KeysEncrypt{
			T: round.T[i],
		}
		keys[i] = keySlc
	}

	db := services.DispatchBuilder{BatchSize: round.BatchSize, Keys: &keys, Output: &om, G: g}

	return &db

}

// Multiplies in the ReceptionKey and the node’s cypher key
func (e Encrypt) Run(g *cyclic.Group, in *SlotEncryptIn, out *SlotEncryptOut, keys *KeysEncrypt) services.Slot {

	// Create Temporary variable
	tmp := cyclic.NewMaxInt()

	// Eq 6.6
	g.Mul(in.ReceptionKey, keys.T, tmp)

	// Eq 6.6
	g.Mul(in.EncryptedMessage, tmp, out.EncryptedMessage)

	// Pass through RecipientID
	out.RecipientID = in.RecipientID

	return out

}
