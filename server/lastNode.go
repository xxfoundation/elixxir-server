package server

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"sync"
)

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
