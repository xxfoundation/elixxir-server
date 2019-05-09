package server

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/server/server/round"
	"sync"
)

type firstNode struct {
	once          sync.Once
	newBatchQueue chan *mixmessages.Batch
	// This struct handles rounds that have finished precomputation and are
	// ready to run realtime
	readyRounds RoundBuffer
}

type RoundBuffer struct {
	CompletedPrecomputations chan *round.Round
	// Whenever a round enters the precomp queue, broadcast this Cond to let
	// waiting nodes know immediately that there's a precomputation available
	CompletedPrecompWait chan *FailableNotify
}

func (fn *firstNode) Initialize() {
	fn.once.Do(func() {
		fn.newBatchQueue = make(chan *mixmessages.Batch, 10)
		fn.readyRounds = RoundBuffer{
			CompletedPrecomputations: make(chan *round.Round, 10),
			CompletedPrecompWait:     make(chan *FailableNotify, 10),
		}
	})
}

type FailableNotify struct {
	Notify chan struct{}
	Valid  bool
	sync.Mutex
}

func (fn *firstNode) GetNewBatchQueue() chan *mixmessages.Batch {
	return fn.newBatchQueue
}

func (fn *firstNode) GetCompletedPrecomps() *RoundBuffer {
	return &fn.readyRounds
}

// Completes the precomputation for a round, and notifies someone who's waiting
func (r *RoundBuffer) CompletePrecomp(precomputedRound *round.Round) {
	// See if there's anyone waiting
	var notify *FailableNotify
	var doneLooking bool
	for !doneLooking {
		select {
		case notify = <-r.CompletedPrecompWait:
			notify.Lock()
			if notify.Valid {
				// This is the waiting RPC we'll notify
				doneLooking = true
				defer notify.Unlock()
			} else {
				// We need to keep looking
				notify.Unlock()
			}
		default:
			// There are no more potential waiting RPCs, so we'll just add the
			// round to the buffer
			doneLooking = true
		}
	}

	// Add the round to the buffer
	r.CompletedPrecomputations <- precomputedRound
	// Notify the waiting RPC, if we got one
	if notify != nil {
		notify.Notify <- struct{}{}
	}
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
