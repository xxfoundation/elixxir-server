////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/server/internal/phase"
)

// getPhaseActivity determines what current.Activity the given phase.Type needs to wait for
func getPhaseActivity(p phase.Type) current.Activity {
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
