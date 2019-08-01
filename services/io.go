////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package services

import (
	"sync"
)

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

func (mi *moduleInput) open(size uint32) {
	mi.input = make(IO_Notify, size)
}
