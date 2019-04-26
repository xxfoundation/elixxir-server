package phase

import (
	"sync"
	"sync/atomic"
)

type StateGroup struct {
	states []*uint32
	phaseIndex  *uint32
	// Types that exist in the state group
	phaseLookup []Type
	rw     sync.RWMutex
}

func (sg *StateGroup) GetState(index int) State {
	sg.rw.RLock()
	defer sg.rw.RUnlock()
	return State(atomic.LoadUint32(sg.states[index]))
}

func (sg *StateGroup) GetCurrentPhase() Type {
	sg.rw.RLock()
	defer sg.rw.RUnlock()
	return phaseLookup[phaseIndex]
}

func (sg *StateGroup) newState(t Type) (int, *uint32) {
	sg.rw.Lock()
	defer sg.rw.Unlock()
	state := uint32(Initialized)
	sg.states = append(sg.states, &state)
	sg.phaseLookup = append(sg.phaseLookup, t)
	return len(sg.states) - 1, &state
}
