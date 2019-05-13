package io

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/server/server"
	"time"
)

// gateway.go is for gateway<->node comms

func GetRoundBufferInfo(roundBuffer *server.PrecompBuffer,
	timeout time.Duration) (int, error) {
	numRounds := len(roundBuffer.CompletedPrecomputations)
	if numRounds != 0 {
		// There are rounds ready, so return
		// Note: This should be considered an edge case
		return len(roundBuffer.CompletedPrecomputations), nil
	} else {
		// Wait for a round to be pushed
		select {
		case <-roundBuffer.PushSignal:
			// Succeed
			return len(roundBuffer.CompletedPrecomputations), nil
		case <-time.After(timeout):
			// Timeout and fail
			return len(roundBuffer.CompletedPrecomputations), errors.New("round buffer is empty")
		}
	}
}

// Returns a completed batch, or waits for a small amount of time for one to
// materialize if there isn't one ready
func GetCompletedBatch(completedRounds chan *mixmessages.Batch,
	timeout time.Duration) (*mixmessages.Batch, error) {
	select {
	case round := <-completedRounds:
		return round, nil
	case <-time.After(timeout):
		return nil, errors.New("No completed batches before the timeout")
	}
}
