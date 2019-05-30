package phase

import (
	"gitlab.com/elixxir/server/services"
	"time"
)

//
type Definition struct {
	Graph               *services.Graph
	Type                Type
	TransmissionHandler Transmit
	Timeout             time.Duration
	DoVerification      bool
}
