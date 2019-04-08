////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package services

import (
	"gitlab.com/elixxir/crypto/cryptops"
	"sync"
)

type OutputNotify chan Chunk

type adapter func(stream Stream, cryptop cryptops.Cryptop, chunk Chunk) error

type Module struct {
	//Public
	// Its method should probably not be called Adapt, I think
	Adapt   adapter
	Cryptop cryptops.Cryptop

	AssignmentSize uint32
	ChunkSize      uint32

	Name string

	NumThreads uint32

	state moduleState

	//Private
	input         OutputNotify
	inputClosed   bool
	inputLock     sync.Mutex
	id            uint64
	inputModules  []*Module
	outputModules []*Module

	assignmentList assignmentList
}

func (m *Module) closeInput() {
	m.inputLock.Lock()
	if !m.inputClosed {
		// Commenting this does prevent the send on closed channel, but also causes the program to not terminate
		close(m.input)
	}
	m.inputLock.Unlock()
}
