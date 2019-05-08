package io

import (
	"gitlab.com/elixxir/server/server/round"
	"testing"
	"time"
)

func TestGetRoundBufferInfo(t *testing.T) {
	// Normal case: length is greater than zero
	c := make(chan *round.Round, 1)
	// Not actually making a Round for concision
	c<-nil
	availableRounds, err := GetRoundBufferInfo(c, time.Second)
	if err != nil {
		t.Error(err)
	}
	if availableRounds != 1 {
		t.Error("Expected 1 round to be available in the buffer")
	}
}
