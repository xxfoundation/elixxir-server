// Implements the Realtime Decrypt phase

package realtime

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/node"
	"gitlab.com/privategrity/server/services"
)

// Decrypt phase completely removes the encryption added by the sending client,
// but the message does not become decrypted because another internal permutation is added
type Decrypt struct{}

// SlotDecryptIn is used to pass external data into Decrypt
type SlotDecryptIn struct {
	//Slot Number of the Data
	slot uint64
	// Eq 3.5
	EncryptedMessage *cyclic.Int
	// Eq 3.7
	EncryptedRecipientID *cyclic.Int
	// Eq 3.5/7
	TransmissionKey *cyclic.Int
	// Pass through
	SenderID *cyclic.Int
}

// SlotDecryptOut is used to pass the results out of Decrypt
type SlotDecryptOut struct {
	//Slot Number of the Data
	slot uint64
	// Eq 3.6
	EncryptedMessage *cyclic.Int
	// Eq 3.8
	EncryptedRecipientID *cyclic.Int
	// Pass through
	SenderID *cyclic.Int
}

// SlotID Returns the Slot number
func (e *SlotDecryptIn) SlotID() uint64 {
	return e.slot
}

// SlotID Returns the Slot number
func (e *SlotDecryptOut) SlotID() uint64 {
	return e.slot
}

// KeysDecrypt holds the keys used by the Decrypt Operation
type KeysDecrypt struct {
	// Eq 3.5: First Unpermuted Internode Message Key
	R *cyclic.Int
	// Eq 3.7: Unpermuted Internode Recipient Key
	U *cyclic.Int
}

// Allocated memory and arranges key objects for the Realtime Decrypt Phase
func (d Decrypt) Build(g *cyclic.Group, face interface{}) *services.DispatchBuilder {

	// Get round from the empty interface
	round := face.(*node.Round)

	// Allocate Memory for output
	om := make([]services.Slot, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = &SlotDecryptOut{
			slot:                 i,
			EncryptedMessage:     cyclic.NewMaxInt(),
			EncryptedRecipientID: cyclic.NewMaxInt(),
			SenderID:             cyclic.NewMaxInt(),
		}
	}

	keys := make([]services.NodeKeys, round.BatchSize)

	// Link the keys for decryption
	for i := uint64(0); i < round.BatchSize; i++ {
		keySlc := &KeysDecrypt{
			R: round.R[i],
			U: round.U[i],
		}
		keys[i] = keySlc
	}

	db := services.DispatchBuilder{BatchSize: round.BatchSize, Keys: &keys, Output: &om, G: g}

	return &db

}

// Removes the encryption added by the Client while simultaneously
// encrypting the message with unpermuted internode keys.
func (d Decrypt) Run(g *cyclic.Group, in *SlotDecryptIn, out *SlotDecryptOut, keys *KeysDecrypt) services.Slot {

	// Create Temporary variable
	tmp := cyclic.NewMaxInt()

	// Eq 3.1
	g.Mul(in.TransmissionKey, keys.R, tmp)

	// Eq 3.1
	g.Mul(in.EncryptedMessage, tmp, out.EncryptedMessage)

	// Eq 3.3
	g.Mul(in.TransmissionKey, keys.U, tmp)

	// Eq 3.3
	g.Mul(in.EncryptedRecipientID, tmp, out.EncryptedRecipientID)

	// Pass through SenderID
	out.SenderID = in.SenderID

	return out

}
