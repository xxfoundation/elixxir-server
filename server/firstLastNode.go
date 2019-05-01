package server

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/server/server/round"
	"sync"
)

type firstNode struct {
	once                  sync.Once
	newBatchQueue         chan *mixmessages.Batch
	completedPrecompQueue chan *round.Round
}

func (fn *firstNode) Initialize() {
	fn.once.Do(func() {
		fn.newBatchQueue = make(chan *mixmessages.Batch, 10)
		fn.completedPrecompQueue = make(chan *round.Round, 10)
	})
}

func (fn *firstNode) GetNewBatchQueue() chan *mixmessages.Batch {
	return fn.newBatchQueue
}

func (fn *firstNode) GetCompletedPrecompQueue() chan *round.Round {
	return fn.completedPrecompQueue
}

type lastNode struct {
	once                sync.Once
	completedBatchQueue chan *mixmessages.Batch
}

func (ln *lastNode) Initialize() {
	ln.once.Do(func() {
		ln.completedBatchQueue = make(chan *mixmessages.Batch, 10)
	})
}

func (ln *lastNode) GetCompletedBatchQueue() chan *mixmessages.Batch {
	return ln.completedBatchQueue
}
