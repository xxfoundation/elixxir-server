package io

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/cryptops/realtime"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
	"strconv"
)

// Blank struct for implementing services.BatchTransmission
type RealtimeIdentifyHandler struct{}

// TransmissionHandler for RealtimeIdentifyMessages
func (h RealtimeIdentifyHandler) Handler(
	roundId string, batchSize uint64, slots []*services.Slot) {
	jww.INFO.Println("Beginning RealtimeEncrypt Phase...")
	// Get round and channel
	round := globals.GlobalRoundMap.GetRound(roundId)
	encryptChannel := round.GetChannel(globals.REAL_ENCRYPT)
	// Create the SlotEncryptIn for sending into RealtimeEncrypt
	for i := uint64(0); i < batchSize; i++ {
		out := (*slots[i]).(*realtime.SlotIdentify)
		// Convert to SlotEncryptIn
		rID, _ := strconv.ParseUint(out.EncryptedRecipientID.Text(10), 10, 64)
		var slot services.Slot = &realtime.SlotEncryptIn{
			Slot:             out.Slot,
			RecipientID:      rID,
			EncryptedMessage: round.LastNode.EncryptedMessage[i],
			ReceptionKey:     cyclic.NewInt(1),
		}
		// Pass slot as input to Encrypt's channel
		encryptChannel <- &slot
	}
}
