////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package phase

// definition.go contains the phase.Definition object

import (
	"gitlab.com/elixxir/server/services"
	"time"
)

// Definition of a phase object, including the phase type, handler and graph
type Definition struct {
	Graph               *services.Graph
	Alternate           func()
	Type                Type
	TransmissionHandler Transmit
	Timeout             time.Duration
	DoVerification      bool
}
