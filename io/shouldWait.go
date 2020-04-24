////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/server/internal/phase"
)

func shouldWait(p phase.Type) current.Activity {
	if p == phase.PrecompShare || p == phase.PrecompGeneration ||
		p == phase.PrecompDecrypt || p == phase.PrecompReveal ||
		p == phase.PrecompPermute {
		return current.PRECOMPUTING
	} else if p == phase.RealDecrypt || p == phase.RealPermute {
		return current.REALTIME
	} else {
		return current.ERROR
	}
}
