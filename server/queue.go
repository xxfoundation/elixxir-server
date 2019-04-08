package server

import (
	"gitlab.com/elixxir/server/node"
	"gitlab.com/elixxir/server/services"
)

type ResourceQueue map[QueueFingerprint]*queueElement

// We gotta come up with a better name for this...
func (rq *ResourceQueue) Leap() {
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

type SendToGraph func(g *services.Graph)

func (rq *ResourceQueue) ProcessIncoming(id node.RoundID, p node.Phase, s2g SendToGraph) bool {
	gf := makeGraphFingerprint(id, p)
	ge, ok := (*rq)[gf]

	if !ok {
		return false
	}

	go func() {
		ge.Lock()
		s2g(ge.g)
	}()

	return true
}

func (rq *ResourceQueue) Push(rid node.RoundID, p node.Phase, g *services.Graph) {
	ge := queueElement{
		g:     g,
		phase: p,
		loc:   len(*rq) - 1,
	}
	gf := makeGraphFingerprint(rid, p)
	ge.Lock()
	(*rq)[gf] = &ge
}
