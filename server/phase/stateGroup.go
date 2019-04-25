package phase

import (
	"sync"
	"sync/atomic"
)

type StateGroup struct {
	states []*uint32
	phase  *uint32
	rw     sync.RWMutex
}

func NewStateGroup() *StateGroup {
	phase := uint32(PRECOMP_GENERATION)
	return &StateGroup{phase: &phase}
}

func (sg *StateGroup) GetState(index int) State {
	sg.rw.RLock()
	defer sg.rw.RUnlock()
	return State(atomic.LoadUint32(sg.states[index]))
}

func (sg *StateGroup) GetCurrentPhase() Type {
	sg.rw.RLock()
	defer sg.rw.RUnlock()
	return Type(atomic.LoadUint32(sg.phase))
}

func (sg *StateGroup) newState() (int, *uint32) {
	sg.rw.Lock()
	defer sg.rw.Unlock()
	state := uint32(Initialized)
	sg.states = append(sg.states, &state)
	return len(sg.states) - 1, &state
}
