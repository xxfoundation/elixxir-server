package services

import "sync"

type IO_Notify chan Chunk

type moduleInput struct {
	input  IO_Notify
	closed bool
	sync.Mutex
}

func (mi *moduleInput) closeInput() {
	mi.Lock()
	if !mi.closed {
		close(mi.input)
	}
	mi.Unlock()
}

func (mi *moduleInput) open() {
	mi.input = make(IO_Notify)
}
