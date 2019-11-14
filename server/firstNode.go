package server

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server/round"
	"sync"
	"testing"
	"time"
)

type firstNode struct {
	once          sync.Once
	runOnce       sync.Once
	newBatchQueue chan *mixmessages.Batch
	// This struct handles rounds that have finished precomputation and are
	// ready to run realtime
	readyRounds *PrecompBuffer

	finishedRound chan id.Round

	currentRoundID id.Round
}

// RoundCreationTransmitter is a function type which is used to notify all nodes
// to create the round
type RoundCreationTransmitter func(*node.Comms, *connect.Circuit, id.Round) error

// RoundStarter is a function type which is used to start a round locally
type RoundStarter func(instance *Instance, roundID id.Round) error

// RunFirstNode is a long running process on the first node which creates new rounds at
// the correct time. It can only be called once. It is passed a function through
// which to interface with the network
func (fn *firstNode) RunFirstNode(instance *Instance, fullRoundTimeout time.Duration,
	transmitter RoundCreationTransmitter, starter RoundStarter) {
	fn.runOnce.Do(
		func() {
			go func() {
				for {
					fn.roundCreationRunner(instance, fullRoundTimeout,
						transmitter, starter)
				}
			}()
		},
	)
}

// roundCreationRunner is a long running process on the first node which
// creates new rounds at the correct time. It is passed a function through
// which to interface with the network
func (fn *firstNode) roundCreationRunner(instance *Instance, fullRoundTimeout time.Duration,
	transmitter RoundCreationTransmitter, starter RoundStarter) {

	err := transmitter(instance.GetNetwork(), instance.GetTopology(), fn.currentRoundID)

	// TODO: proper error handling will broadcast a signal to killChan to round,
	// allowing to continue
	if err != nil {
		jww.FATAL.Panicf("Round failed to create round remotely: %+v", err)
	}

	//start the round locally
	err = starter(instance, fn.currentRoundID)
	if err != nil {
		jww.FATAL.Panicf("Round failed to start round locally: %+v", err)
	}

	select {
	case finishedRound := <-fn.finishedRound:
		//TODO: proper error handling
		if finishedRound != fn.currentRoundID {
			jww.FATAL.Panicf("Incorrect Round finished; Expected: "+
				"%v, Recieved: %v", fn.currentRoundID, finishedRound)
		}

		go func() {
			errMetric := instance.definition.MetricsHandler(instance, finishedRound)
			if errMetric != nil {
				jww.ERROR.Printf("Failure in posting metrics for round %d: %v",
					finishedRound, err)
			}
		}()
	case <-time.After(fullRoundTimeout):
		//TODO: proper error handling
		jww.FATAL.Panicf("Round did not finish within timeout of %v",
			fullRoundTimeout)
	}

	fn.currentRoundID++
}

// Initialize populates the first node structure properly.  It can only be called
// once.
func (fn *firstNode) Initialize() {
	fn.once.Do(func() {
		fn.newBatchQueue = make(chan *mixmessages.Batch, 10)
		fn.readyRounds = &PrecompBuffer{
			CompletedPrecomputations: make(chan *round.Round, 10),
			// The buffer size on the push signal must be 1 for correctness
			//PushSignal: make(chan struct{}, 1),
			PushSignal: make(chan struct{}),
		}
		fn.finishedRound = make(chan id.Round, 1)
		fn.currentRoundID = 0
	})
}

// GetNewBatchQueue returns the que which stores new batches
func (fn *firstNode) GetNewBatchQueue() chan *mixmessages.Batch {
	return fn.newBatchQueue
}

// GetCompletedPrecomps returns the PrecompBuffer which is used to track and
// signal the completion of precomputations
func (fn *firstNode) GetCompletedPrecomps() *PrecompBuffer {
	return fn.readyRounds
}

// FinishRound is used to denote a round, by its id, is completed
func (fn *firstNode) FinishRound(id id.Round) {
	fn.finishedRound <- id
}

// GetFinishedRounds returns the finished rounds channel just for testing
func (fn *firstNode) GetFinishedRounds(t *testing.T) chan id.Round {
	if t == nil {
		jww.FATAL.Panicf("GetFinishedRounds can only be used in tests")
	}
	return fn.finishedRound
}

type PrecompBuffer struct {
	CompletedPrecomputations chan *round.Round
	// Whenever a round gets Pushed, this channel gets signaled
	PushSignal chan struct{}
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
