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
	timeoutPrecomputation(round, time.Nanosecond)
	time.Sleep(time.Second)
	if round.GetPhase() != globals.ERROR {
		t.Error("Precomputation: Round didn't time out")
	}

	globals.ResetRound(round)
	timeoutRealtime(round, time.Nanosecond)
	time.Sleep(time.Second)
	if round.GetPhase() != globals.ERROR {
		t.Error("Realtime: Round didn't time out")
	}
}

func TestNotTimeoutRound(t *testing.T) {
	round := globals.NewRound(1)
	timeoutRealtime(round, time.Minute)
	time.Sleep(time.Second)
	if round.GetPhase() == globals.ERROR {
		t.Error("Realtime: Round timed out")
	}

	globals.ResetRound(round)
	timeoutPrecomputation(round, time.Minute)
	time.Sleep(time.Second)
	if round.GetPhase() == globals.ERROR {
		t.Error("Precomputation: Round timed out")
	}
}
