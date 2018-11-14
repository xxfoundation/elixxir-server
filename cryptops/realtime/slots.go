package realtime

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/id"
)

// Slot is a general slot structure used by all other
// realtime cryptops. The semantics of each element change and not
// all elements are used by every cryptop, but the purpose remains the same
// as the data travels through realtime.
type Slot struct {
	Slot uint64
	// Encrypted RecipientID
	EncryptedRecipient *cyclic.Int
	// Encrypted or plaintext Message
	Message *cyclic.Int
	// Plaintext SenderID or RecipientID
	CurrentID *id.UserID
	// TransmissionKey, ReceptionKey, etc
	CurrentKey *cyclic.Int
	// Salt for client operations (only for Decrypt and Encrypt Phases)
	Salt []byte
}

// SlotID functions return the Slot number
func (e *Slot) SlotID() uint64 {
	return e.Slot
}
