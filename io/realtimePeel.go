package io

import (
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/server/cryptops/realtime"
	"gitlab.com/privategrity/server/globals"
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
		globals.Users.GetUser(slot.RecipientID).MessageBuffer <- pb.CmixMessage{
			slot.EncryptedMessage.Bytes(), make([]byte, 0)}
	}
	jww.INFO.Println("Realtime Finished!")
}
