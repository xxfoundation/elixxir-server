package services

import (
	"gitlab.com/elixxir/server/globals"
	"sync"
)

type graphElement struct {
	g   *Graph
	loc int
	sync.Mutex
}

type ResourceQueue map[GraphFingerprint]*graphElement

// We gotta come up with a better name for this...
func (rq *ResourceQueue) Leap() {
	for _, g := range *rq {
		g.loc--

		switch g.loc {
		case -1:
			delete(*rq, g.g.GetFingerprint())
		case 0:
			g.Unlock()
		case 1:
			g.g.Run()
		}
	}
}

type SendToGraph func(g *Graph)

func (rq *ResourceQueue) ProcessIncoming(id globals.RoundID, p globals.Phase, s2g SendToGraph) bool {
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

func (rq *ResourceQueue) Push(rid globals.RoundID, p globals.Phase, g *Graph) {
	ge := graphElement{
		g:   g,
		loc: len(*rq),
	}
	gf := makeGraphFingerprint(rid, p)
	ge.Lock()
	(*rq)[gf] = &ge
}
