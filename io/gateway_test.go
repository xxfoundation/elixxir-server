package io

import (
	"gitlab.com/elixxir/server/server/round"
	"testing"
	"time"
)

// Shows that GetRoundBufferInfo timeout works as intended
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

	// More than timeout case: length is zero and stays there
	c = make(chan *round.Round, 1)
	_, err = GetRoundBufferInfo(c, 200*time.Millisecond)
	if err == nil {
		t.Error("Round buffer info timeout should have resulted in an error")
	}

    // Less than timeout case: length that's zero, then one, should result in
    // a resulting length of one
    c = make(chan *round.Round, 1)
    before := time.Now()
    time.AfterFunc(200*time.Millisecond, func() {
        c<-nil
	})
    availableRounds, err = GetRoundBufferInfo(c, time.Second)
    // elapsed time should be around 200 milliseconds,
    // because that's when the channel write happened
    after := time.Since(before)
    if after < 100*time.Millisecond || after > 400 *time.Millisecond {
    	t.Errorf("RoundBufferInfo result came in at an odd duration: %v", after)
	}
	if err != nil {
		t.Error(err)
	}
    if availableRounds!= 1 {
		t.Error("Expected 1 round to be available in the buffer")
	}
}
