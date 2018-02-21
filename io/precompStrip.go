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
	jww.INFO.Println("Precomputation Finished!")
	// Since precomputation is finished, we put this round's ID on the
	// channel of waiting rounds so that the round can wait until we've
	// received enough messages to kick off realtime
	globals.PutWaitingRoundID(roundId)
}
