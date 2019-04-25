package round

import (
	"gitlab.com/elixxir/primitives/id"
	"sync"
)

type Manager struct {
	roundMap *sync.Map
}

func NewManager() *Manager {
	rmap := sync.Map{}
	return &Manager{&rmap}
}

func (rm *Manager) AddRound(round *Round) {
	rm.roundMap.Store(round.id, round)
}

func (rm *Manager) GetRound(id id.Round) *Round {
	r, ok := rm.roundMap.Load(id)

	if !ok {
		return nil
	}

	return r.(*Round)
}

// Deletes the round for this ID from the manager, if the manager is keeping
// track of it
func (rm *Manager) DeleteRound(id id.Round) {
	rm.roundMap.Delete(id)
}
