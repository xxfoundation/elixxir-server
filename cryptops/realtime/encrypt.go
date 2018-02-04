// Implements the Realtime Encrypt phase

package realtime

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/cryptops"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
)

// The Encrypt phase adds in the final internode keys while simultaneously
// adding in Reception Keys for the recipient.
type Encrypt struct{}

// SlotEncrypt is used to pass external data into Encrypt
type SlotEncryptIn struct {
	// Slot Number of the Data
	Slot uint64
	// ID of the client who will recieve the message (Pass through)
	RecipientID uint64
	// Permuted Message Encrypted with R and S and some Ts and Reception Keys
	EncryptedMessage *cyclic.Int
	// Shared Key between the client who receives the message and the node
	ReceptionKey *cyclic.Int
}

// SlotEncryptOut is used to pass  the results out of Encrypt
type SlotEncryptOut struct {
	//Slot Number of the Data
	Slot uint64
	// ID of the client who will recieve the message (Pass through)
	RecipientID uint64
	// Permuted Message Encrypted with R and S and some Ts and Reception Keys
	EncryptedMessage *cyclic.Int
}

// SlotID Returns the Slot number
func (e *SlotEncryptIn) SlotID() uint64 {
	return e.Slot
}

// ID of the user for keygen
func (e *SlotEncryptIn) UserID() uint64 {
	return e.RecipientID
}

// Cyclic int to place the key in
func (e *SlotEncryptIn) Key() *cyclic.Int {
	return e.ReceptionKey
}

// Returns the KeyType
func (e *SlotEncryptIn) GetKeyType() cryptops.KeyType {
	return cryptops.RECEPTION
}

// SlotID Returns the Slot number
func (e *SlotEncryptOut) SlotID() uint64 {
	return e.Slot
}

// KeysEncrypt holds the keys used by the Encrypt Operation
type KeysEncrypt struct {
	// Eq 6.6: Second Unpermuted Internode Message Key
	T *cyclic.Int
}

// Allocated memory and arranges key objects for the Realtime Encrypt Phase
func (e Encrypt) Build(g *cyclic.Group, face interface{}) *services.DispatchBuilder {

	// Get round from the empty interface
	round := face.(*globals.Round)

	// Allocate Memory for output
	om := make([]services.Slot, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = &SlotEncryptOut{
			Slot:             i,
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

	// Eq 6.6: Multiplies the Reception Key and the Second Unpermuted
	// Internode Keys into the Encrypted Message
	g.Mul(in.ReceptionKey, keys.T, tmp)
	g.Mul(in.EncryptedMessage, tmp, out.EncryptedMessage)

	// Pass through RecipientID
	out.RecipientID = in.RecipientID

	return out

}
