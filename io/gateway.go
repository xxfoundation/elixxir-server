package io

import (
	"github.com/pkg/errors"
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
