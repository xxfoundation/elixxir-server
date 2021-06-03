///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package phase

// definition.go contains the phase.Definition object

import (
	"gitlab.com/elixxir/server/services"
	"time"
)

// The definition of a phase object, including the
// phase type, handler and graph
type Definition struct {
	Graph               *services.Graph
	Alternate           func()
	Type                Type
	TransmissionHandler Transmit
	Timeout             time.Duration
	DoVerification      bool
}
