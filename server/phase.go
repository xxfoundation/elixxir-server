package server

import (
	"gitlab.com/elixxir/server/node"
	"gitlab.com/elixxir/server/services"
	"sync"
)

type Phase struct {
	g                   *services.Graph
	round               *Round
	phase               node.PhaseType
	transmissionHandler Transmission
	sync.Mutex
}

func (qe *Phase) GetFingerprint() PhaseFingerprint {
	return makeGraphFingerprint(qe.round.GetID(), qe.phase)
}
