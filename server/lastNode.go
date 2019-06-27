package server

import (
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/services"
	"sync"
)

type LastNode struct {
	once                sync.Once
	completedBatchQueue chan *CompletedRound
}

type CompletedRound struct {
	RoundID    id.Round
	Receiver   chan services.Chunk
	GetMessage phase.GetMessage
}

func (cr *CompletedRound) GetChunk() (services.Chunk, bool) {
	chunk, ok := <-cr.Receiver
	return chunk, ok
}

func (ln *LastNode) Initialize() {
	ln.once.Do(func() {
		ln.completedBatchQueue = make(chan *CompletedRound, 10)
	})
}

func (ln *LastNode) GetCompletedBatchQueue() chan *CompletedRound {
	return ln.completedBatchQueue
}

func (ln *LastNode) SendCompletedBatchQueue(cr CompletedRound) {
	ln.completedBatchQueue <- &cr
}
