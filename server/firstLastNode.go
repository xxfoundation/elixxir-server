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
	readyRounds *PrecompBuffer
}

type PrecompBuffer struct {
	CompletedPrecomputations chan *round.Round
	// Whenever a round gets Pushed, this channel gets signaled
	PushSignal chan struct{}
}

func (fn *firstNode) Initialize() {
	fn.once.Do(func() {
		fn.newBatchQueue = make(chan *mixmessages.Batch, 10)
		fn.readyRounds = &PrecompBuffer{
			CompletedPrecomputations: make(chan *round.Round, 10),
			// The buffer size on the push signal must be 1 for correctness
			//PushSignal: make(chan struct{}, 1),
			PushSignal: make(chan struct{}),
		}
	})
}

func (fn *firstNode) GetNewBatchQueue() chan *mixmessages.Batch {
	return fn.newBatchQueue
}

func (fn *firstNode) GetCompletedPrecomps() *PrecompBuffer {
	return fn.readyRounds
}

// Completes the precomputation for a round, and notifies someone who's waiting
func (r *PrecompBuffer) Push(precomputedRound *round.Round) {
	// Add the round to the buffer
	r.CompletedPrecomputations <- precomputedRound

	// Notify the waiting RPC, if there is one
	select {
	case r.PushSignal <- struct{}{}:
	default:
	}
}

// Return the next round in the buffer, if it exists
// Does not block
// Receiving with `, ok` determines whether the channel has been closed or
// not, not whether there are items available on the channel.
// So, to return false if there wasn't something on the channel, we need
// to select.
func (r *PrecompBuffer) Pop() (*round.Round, bool) {
	select {
	case precomputedRound := <-r.CompletedPrecomputations:
		return precomputedRound, true
	default:
		return nil, false
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
