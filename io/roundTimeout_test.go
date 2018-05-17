////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"testing"
	"gitlab.com/privategrity/server/globals"
	"time"
)

func TestTimeoutRound(t *testing.T) {
	round := globals.NewRound(1)
	roundId := "neil"
	globals.GlobalRoundMap = globals.NewRoundMap()
	globals.GlobalRoundMap.AddRound(roundId, round)
	timeoutPrecomputation(roundId, time.Nanosecond)
	time.Sleep(time.Second)
	if round.GetPhase() != globals.ERROR {
		t.Error("Precomputation: Round didn't time out")
	}

	globals.ResetRound(round)
	roundId2 := "neal"
	timeoutRealtime(roundId2, time.Nanosecond)
	time.Sleep(time.Second)
	if round.GetPhase() != globals.ERROR {
		t.Error("Realtime: Round didn't time out")
	}
}

func TestNotTimeoutRound(t *testing.T) {
	round := globals.NewRound(1)
	roundId := "neil"
	globals.GlobalRoundMap = globals.NewRoundMap()
	globals.GlobalRoundMap.AddRound(roundId, round)
	timeoutRealtime(roundId, time.Minute)
	time.Sleep(time.Second)
	if round.GetPhase() == globals.ERROR {
		t.Error("Realtime: Round timed out")
	}

	globals.ResetRound(round)
	roundId2 := "neal"
	timeoutPrecomputation(roundId2, time.Minute)
	time.Sleep(time.Second)
	if round.GetPhase() == globals.ERROR {
		t.Error("Precomputation: Round timed out")
	}
}
