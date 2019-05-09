package io

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/server/server"
	"time"
)

// gateway.go is for gateway<->node comms

func GetRoundBufferInfo(roundBuffer *server.RoundBuffer,
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
