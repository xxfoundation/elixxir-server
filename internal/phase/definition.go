////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
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
	Type                Type
	TransmissionHandler Transmit
	Timeout             time.Duration
	DoVerification      bool
}
