////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package io

import (
	"gitlab.com/elixxir/server/cryptops/precomputation"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/services"

	jww "github.com/spf13/jwalterweatherman"
	"time"
)

// Blank struct for implementing services.BatchTransmission
type PrecompStripHandler struct{}

// TransmissionHandler for PrecompStripMessages
func (h PrecompStripHandler) Handler(
	roundId string, batchSize uint64, slots []*services.Slot) {
	startTime := time.Now()
	jww.INFO.Printf("Starting PrecompStrip.Handler(RoundId: %s) at %s",
		roundId, startTime.Format(time.RFC3339))

	elapsed := startTime.Sub(globals.GlobalRoundMap.GetRound(roundId).
		CryptopStartTimes[globals.PRECOMP_STRIP])

	jww.DEBUG.Printf("PrecompStrip Crypto took %v ms for "+
		"RoundId %s", elapsed, roundId)

	round := globals.GlobalRoundMap.GetRound(roundId)
	if round == nil {
		jww.INFO.Printf("skipping round %s, because it's dead", roundId)
		return
	}

	// Retrieve the Precomputations
	for i := uint64(0); i < batchSize; i++ {
		slot := (*slots[i]).(*precomputation.PrecomputationSlot)
		// Save each LastNode Precomputation
		round.LastNode.MessagePrecomputation[i] = slot.MessagePrecomputation
		round.LastNode.AssociatedDataPrecomputation[i] = slot.AssociatedDataPrecomputation
		jww.DEBUG.Printf("MessagePrecomputation Result: %v",
			slot.MessagePrecomputation.Text(10))
		jww.DEBUG.Printf("AssociatedDataPrecomputation Result: %v",
			slot.AssociatedDataPrecomputation.Text(10))
	}

	// Advance internal state to PRECOMP_DECRYPT (the next phase)
	globals.GlobalRoundMap.SetPhase(roundId, globals.PRECOMP_COMPLETE)

	endTime := time.Now()
	jww.INFO.Printf("Finished PrecompStrip.Handler(RoundId: %s) in %d ms",
		roundId, (endTime.Sub(startTime))/time.Millisecond)
	jww.INFO.Printf("Precomputation Finished at %s!", endTime.Format(time.RFC3339))
}
