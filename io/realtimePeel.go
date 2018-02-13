package io

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/privategrity/server/cryptops/realtime"
	"gitlab.com/privategrity/server/services"
)

// Blank struct for implementing services.BatchTransmission
type RealtimePeelHandler struct{}

// TransmissionHandler for RealtimePeelMessages
func (h RealtimePeelHandler) Handler(
	roundId string, batchSize uint64, slots []*services.Slot) {
	// Retrieve the EncryptedMessage
	for i := uint64(0); i < batchSize; i++ {
		slot := (*slots[i]).(*realtime.SlotPeel)
		jww.DEBUG.Printf("EncryptedMessage Result: %s",
			slot.EncryptedMessage.Bytes())
	}
	jww.INFO.Println("Realtime Finished!")
}
