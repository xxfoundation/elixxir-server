///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package io

import (
	"git.xx.network/elixxir/primitives/current"
	"git.xx.network/elixxir/server/internal/phase"
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
