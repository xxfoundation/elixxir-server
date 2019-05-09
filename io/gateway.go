package io

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/round"
	"time"
)

// gateway.go is for gateway<->node comms

func GetRoundBufferInfo(roundBuffer chan *round.Round, timeout time.Duration) (int,
	error) {
	c := make(chan struct{}, 1)
	go func() {
		for len(roundBuffer) == 0 {
			// This amount of time should allow us to check the round buffer
			// five times given the current 1-second timeout
			time.Sleep(192 * time.Millisecond)
		}
		c <- struct{}{}
	}()
	select {
	case <-c:
		return len(roundBuffer), nil
	case <-time.After(timeout):
		return len(roundBuffer), errors.New("round buffer is empty")
	}
}

func GetRoundBufferInfo2(roundBuffer *server.RoundBuffer,
	timeout time.Duration) (int, error) {
	numRounds := len(roundBuffer.CompletedPrecomputations)
	if numRounds != 0 {
		// there are rounds ready
		// note: this case is unlikely
		return numRounds, nil
	} else {
		// get in line to be notified when a round is ready
		notifyMe := server.FailableNotify{
			Notify: make(chan struct{}),
			Valid:  true,
		}
		roundBuffer.CompletedPrecompWait <- &notifyMe

		// either time out, or be notified when a round is ready
		select {
		case <-notifyMe.Notify:
			// success: there's now a round available
			return len(roundBuffer.CompletedPrecomputations), nil
		case <-time.After(timeout):
			// timeout and mark the notify channel as no longer valid
			notifyMe.Lock()
			defer notifyMe.Unlock()
			notifyMe.Valid = false
			return len(roundBuffer.CompletedPrecomputations), errors.New("round buffer is empty")
		}
	}
}
