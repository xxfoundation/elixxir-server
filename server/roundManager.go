package server

import (
	"gitlab.com/elixxir/server/node"
	"sync"
)

type RoundManager struct {
	roundMap *sync.Map
}

func (rm *RoundManager) AddRound(round *Round) {
	rm.roundMap.Store(round.id, round)
}

func (rm *RoundManager) GetRound(id node.RoundID) *Round {
	r, ok := rm.roundMap.Load(id)

	if !ok {
		return nil
	}

	return r.(*Round)
}

// Deletes the round for this ID from the manager, if the manager is keeping
// track of it
func (rm *RoundManager) DeleteRound(id nodeRoundID) {
	rm.roundMap.Delete(id)
}