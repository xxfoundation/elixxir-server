package server

import (
	"sync"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/elixxir/server/node"
)

type graphElement struct {
	g   	*services.Graph
	phase 	node.Phase
	loc 	int
	sync.Mutex
}

type ResourceQueue map[GraphFingerprint]*graphElement

// We gotta come up with a better name for this...
func (rq *ResourceQueue) Leap() {
	for _, g := range *rq {
		g.loc--

		switch g.loc {
		case -1:
			delete(*rq, g.phase)
		case 0:
			g.Unlock()
		case 1:
			g.g.Run()
		}
	}
}

type SendToGraph func(g *services.Graph)

func (rq *ResourceQueue) ProcessIncoming(id RoundID, p Phase, s2g SendToGraph) bool {
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

func (rq *ResourceQueue) Push(rid RoundID, p Phase, g *services.Graph) {
	ge := graphElement{
		g:   	g,
		phase: 	p,
		loc: 	len(*rq)-1,
	}
	gf := makeGraphFingerprint(rid, p)
	ge.Lock()
	(*rq)[gf] = &ge
}
