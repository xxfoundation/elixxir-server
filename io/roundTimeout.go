////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/server/globals"
	"time"
)

// Errors a round after a certain time if its precomputation isn't done
func timeoutPrecomputation(roundId string, timeout time.Duration) {
	round := globals.GlobalRoundMap.GetRound(roundId)
	success := false
	timer := time.AfterFunc(timeout, func() {
		if !success && round.GetPhase() < globals.PRECOMP_COMPLETE {
			// Precomp wasn't totally complete before timeout. Set it to error
			jww.ERROR.Printf("Precomputation incomplete: Timing out round %v"+
				" on node %v with phase %v", roundId, *globals.NodeID,
				round.GetPhase().String())
			round.SetPhase(globals.ERROR)
		}
	})
	go func() {
		round.WaitUntilPhase(globals.PRECOMP_COMPLETE)
		jww.INFO.Printf("Waited until phase %v"+
			" on node %v for round %v", round.GetPhase().String(),
			*globals.NodeID,
			roundId)
		success = true
		timer.Stop()
	}()
}

// Errors a round after a certain time if its realtime process isn't done
func timeoutRealtime(roundId string, timeout time.Duration) {
	round := globals.GlobalRoundMap.GetRound(roundId)
	success := false
	timer := time.AfterFunc(timeout, func() {
		if !success && round.GetPhase() < globals.REAL_COMPLETE {
			// Realtime wasn't totally complete before timeout. Set it to error
			jww.ERROR.Printf("Realtime incomplete: Timing out round %v on node"+
				" %v with phase %v", roundId, *globals.NodeID,
				round.GetPhase().String())
			round.SetPhase(globals.ERROR)
		}
	})
	go func() {
		round.WaitUntilPhase(globals.REAL_COMPLETE)
		jww.INFO.Printf("Waited until phase %v"+
			" on node %v for round %v", round.GetPhase().String(),
			*globals.NodeID,
			roundId)
		success = true
		timer.Stop()
	}()
}
