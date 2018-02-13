package io

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/cryptops/precomputation"
	"gitlab.com/privategrity/server/cryptops/realtime"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
)

// Blank struct for implementing services.BatchTransmission
type PrecompStripHandler struct{}

// TransmissionHandler for PrecompStripMessages
func (h PrecompStripHandler) Handler(
	roundId string, batchSize uint64, slots []*services.Slot) {
	round := globals.GlobalRoundMap.GetRound(roundId)
	// Retrieve the Precomputations
	for i := uint64(0); i < batchSize; i++ {
		slot := (*slots[i]).(*precomputation.SlotStripOut)
		// Save each LastNode Precomputation
		round.LastNode.MessagePrecomputation[i] = slot.MessagePrecomputation
		round.LastNode.RecipientPrecomputation[i] = slot.RecipientPrecomputation
		jww.DEBUG.Printf("MessagePrecomputation Result: %v",
			slot.MessagePrecomputation.Text(10))
		jww.DEBUG.Printf("RecipientPrecomputation Result: %v",
			slot.RecipientPrecomputation.Text(10))
	}
	jww.INFO.Println("Precomputation Finished!")

	// Transition to Realtime phase
	StartRealtime(roundId, batchSize)
}

// Begin Realtime once Precomputation is finished
func StartRealtime(roundId string, batchSize uint64) {
	// TODO Place transition logic here
	testText := []byte("Hello, world!")
	jww.DEBUG.Printf("RealtimePeel EncryptedMessage Input: %s", testText)
	inputMsg := services.Slot(&realtime.SlotDecryptOut{
		Slot:                 0,
		SenderID:             1,
		EncryptedMessage:     cyclic.NewIntFromBytes(testText),
		EncryptedRecipientID: cyclic.NewInt(1),
	})

	jww.INFO.Println("Beginning RealtimeDecrypt Phase...")
	KickoffDecryptHandler(roundId, batchSize, []*services.Slot{&inputMsg})
}
