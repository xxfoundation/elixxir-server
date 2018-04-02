////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package io

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/privategrity/server/cryptops/precomputation"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
)

// Blank struct for implementing services.BatchTransmission
type PrecompStripHandler struct{}

// TransmissionHandler for PrecompStripMessages
func (h PrecompStripHandler) Handler(
	roundId string, batchSize uint64, slots []*services.Slot) {
	startTime := time.Now()
	jww.INFO.Printf("Starting PrecompStrip.Handler(RoundId: %s) at %s",
		roundId, startTime.Format(time.RFC3339))

	round := globals.GlobalRoundMap.GetRound(roundId)
	// Retrieve the Precomputations
	for i := uint64(0); i < batchSize; i++ {
		slot := (*slots[i]).(*precomputation.PrecomputationSlot)
		// Save each LastNode Precomputation
		round.LastNode.MessagePrecomputation[i] = slot.MessagePrecomputation
		round.LastNode.RecipientPrecomputation[i] = slot.RecipientIDPrecomputation
		jww.DEBUG.Printf("MessagePrecomputation Result: %v",
			slot.MessagePrecomputation.Text(10))
		jww.DEBUG.Printf("RecipientPrecomputation Result: %v",
			slot.RecipientIDPrecomputation.Text(10))
	}

	endTime := time.Now()
	jww.INFO.Printf("Finished PrecompStrip.Handler(RoundId: %s) in %d ms",
		roundId, (endTime.Sub(startTime))*time.Millisecond)
	jww.INFO.Printf("Precomputation Finished at %s!", endTime.Format(time.RFC3339))
}
