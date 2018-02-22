package realtime

import "gitlab.com/privategrity/crypto/cyclic"

// RealtimeSlot is a general slot structure used by all other
// realtime cryptops. The semantics of each element change and not
// all elements are used by every cryptop, but the purpose remains the same
// as the data travels through realtime.
type RealtimeSlot struct {
	Slot uint64
	// Encrypted RecipientID
	EncryptedRecipient *cyclic.Int
	// Encrypted or plaintext Message
	Message *cyclic.Int
	// Plaintext SenderID or RecipientID
	CurrentID uint64
	// TransmissionKey, ReceptionKey, etc
	CurrentKey *cyclic.Int
}

// SlotID functions return the Slot number
func (e *RealtimeSlot) SlotID() uint64 {
	return e.Slot
}
