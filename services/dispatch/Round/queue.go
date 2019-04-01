package Round

import (
	"encoding/binary"
	"gitlab.com/elixxir/dispatch/dispatch"
	"gitlab.com/elixxir/dispatch/globals"
	"sync"
)

type RoundID uint64

type graphElement struct {
	g   *dispatch.Graph
	loc int
	sync.Mutex
}

type ResourceQueue map[graphFingerprint]*graphElement

// We gotta come up with a better name for this...
func (rq *ResourceQueue) Leap() {
	for _, g := range *rq {
		g.loc--

		switch g.loc {
		case -1:
			delete(*rq, g)
		case 0:
			g.Unlock()
		case 1:
			g.g.Run()
		}
	}
}

type SendToGraph func(g *dispatch.Graph)

func (rq *ResourceQueue) ProcessIncoming(id RoundID, p globals.Phase, s2g SendToGraph) bool {
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

type graphFingerprint [9]byte

func makeGraphFingerprint(rid RoundID, p globals.Phase) graphFingerprint {
	var gf graphFingerprint
	binary.BigEndian.PutUint64(gf[:8], uint64(rid))
	gf[8] = byte(p)
	return gf
}

func (rq *ResourceQueue) Push(rid RoundID, p globals.Phase, g *dispatch.Graph) {
	ge := graphElement{
		g:   g,
		loc: len(*rq),
	}
	gf := makeGraphFingerprint(rid, p)
	ge.Lock()
	(*rq)[gf] = &ge
}
