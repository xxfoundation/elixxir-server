package dispatch

import (
	"sync"
)

type OutputNotify chan Lot

type adapter func(stream Stream, function Function, chunk Lot, callback ErrorCallback)

type Function interface {
	GetFuncName() string
	GetMinSize() uint32
}

// Should probably add more params to this like block ID, worker thread ID, etc
type ErrorCallback func(err error)

type Module struct {
	//Public
	// Its method should probably not be called Adapt, I think
	Adapt adapter
	F     Function

	InputSize uint32

	Name string

	NumThreads uint32

	moduleState

	//Private
	input         OutputNotify
	inputClosed   bool
	inputLock     sync.Mutex
	id            uint64
	inputModules  []*Module
	outputModules []*Module

	assignmentList
}

func (m *Module) closeInput() {
	m.inputLock.Lock()
	if !m.inputClosed {
		// Commenting this does prevent the send on closed channel, but also causes the program to not terminate
		close(m.input)
	}
	m.inputLock.Unlock()
}
