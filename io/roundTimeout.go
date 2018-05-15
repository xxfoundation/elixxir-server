////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/privategrity/server/globals"
	"time"
	jww "github.com/spf13/jwalterweatherman"
)

// Errors a round after a certain time if its precomputation isn't done
func timeoutPrecomputation(roundId string, timeout time.Duration) {
	round := globals.GlobalRoundMap.GetRound(roundId)
	time.AfterFunc(timeout, func() {
		if round.GetPhase() < globals.PRECOMP_COMPLETE {
			// Precomp wasn't totally complete before timeout. Set it to error
			jww.ERROR.Printf("Precomputation incomplete: Timing out round %v" +
				" on node %v", roundId, globals.NodeID(0))
			round.SetPhase(globals.ERROR)
		}
	})
}

// Errors a round after a certain time if its realtime process isn't done
func timeoutRealtime(roundId string, timeout time.Duration) {
	round := globals.GlobalRoundMap.GetRound(roundId)
	time.AfterFunc(timeout, func() {
		if round.GetPhase() < globals.REAL_COMPLETE {
			// Realtime wasn't totally complete before timeout. Set it to error
			jww.ERROR.Printf("Realtime incomplete: Timing out round %v on node"+
				" %v", roundId, globals.NodeID(0))
			round.SetPhase(globals.ERROR)
		}
	})
}
