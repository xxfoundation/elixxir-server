package server

import (
	"gitlab.com/elixxir/server/node"
	"gitlab.com/elixxir/server/services"
	"sync"
)


type roundPhaseStates [node.NUM_PHASES*100]Phase

type ResourceQueue struct {
	phaseLookup sync.Map
	buf roundPhaseStates
	loc *uint64
	sync.RWMutex
}

// We gotta come up with a better name for this...
func (rq *ResourceQueue) Leap() {
	rq.


	rq.phases.Range(func(key,value interface{})bool{
		return true
	})



	for _, g := range *rq {
		g.loc--

		switch g.loc {
		case -1:
			delete(*rq, g.GetFingerprint())
		case 0:
			g.Unlock()
		case 1:
			g.g.Run()
		}
	}
}


func (rq *ResourceQueue) ProcessIncoming(id node.RoundID, p node.PhaseType) bool {
	fingerprint := makeGraphFingerprint(id, p)
	phaseLocInterface, ok := rq.phaseLookup.Load(fingerprint)
	phaseLoc := phaseLocInterface.(uint64)

	if !ok {
		return false
	}

	if

	phase := rq.buf[phaseLoc]

	phase.Lock()



	return true
}

func (rq *ResourceQueue) Push(rid node.RoundID, p node.PhaseType, g *services.Graph) {
	ge := queueElement{
		g:     g,
		phase: p,
		loc:   len(*rq) - 1,
	}
	gf := makeGraphFingerprint(rid, p)
	ge.Lock()
	(*rq)[gf] = &ge
}
