///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////
package state

import (
	"fmt"
)

type Status uint32

const (
	NOT_STARTED = Status(iota)
	STARTED
	ENDED
	NUM_STATUS
)

// Stringer to get the name of the activity, primarily for for error prints
func (s Status) String() string {
	switch s {
	case NOT_STARTED:
		return "NOT_STARTED"
	case STARTED:
		return "STARTED"
	case ENDED:
		return "ENDED"
	default:
		return fmt.Sprintf("UNKNOWN STATE: %d", s)
	}
}
