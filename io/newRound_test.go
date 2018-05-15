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
	timeoutRound(round, time.Nanosecond)
	time.Sleep(time.Second)
	if round.GetPhase() != globals.ERROR {
		t.Error("Round didn't time out")
	}
}

func TestNotTimeoutRound(t *testing.T) {
	round := globals.NewRound(1)
	timeoutRound(round, time.Minute)
	time.Sleep(time.Second)
	if round.GetPhase() == globals.ERROR {
		t.Error("Round timed out")
	}
}
