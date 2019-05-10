package io

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/round"
	"testing"
	"time"
)

// Shows that GetRoundBufferInfo timeout works as intended
func TestGetRoundBufferInfo(t *testing.T) {
	// Normal case: length is greater than zero
	c := &server.PrecompBuffer{
		CompletedPrecomputations: make(chan *round.Round, 1),
		PushSignal:               make(chan struct{}),
	}

	// Not actually making a Round for concision
	c.Push(nil)
	availableRounds, err := GetRoundBufferInfo(c, time.Second)
	if err != nil {
		t.Error(err)
	}
	if availableRounds != 1 {
		t.Error("Expected 1 round to be available in the buffer")
	}

	// More than timeout case: length is zero and stays there
	c = &server.PrecompBuffer{
		CompletedPrecomputations: make(chan *round.Round, 1),
		PushSignal:               make(chan struct{}),
	}
	_, err = GetRoundBufferInfo(c, 200*time.Millisecond)
	if err == nil {
		t.Error("Round buffer info timeout should have resulted in an error")
	}

	// Less than timeout case: length that's zero, then one, should result in
	// a resulting length of one
	c = &server.PrecompBuffer{
		CompletedPrecomputations: make(chan *round.Round, 1),
		PushSignal:               make(chan struct{}),
	}
	before := time.Now()
	time.AfterFunc(200*time.Millisecond, func() {
		c.Push(nil)
	})
	availableRounds, err = GetRoundBufferInfo(c, time.Second)
	// elapsed time should be around 200 milliseconds,
	// because that's when the channel write happened
	after := time.Since(before)
	if after < 100*time.Millisecond || after > 400*time.Millisecond {
		t.Errorf("RoundBufferInfo result came in at an odd duration: %v", after)
	}
	if err != nil {
		t.Error(err)
	}
	if availableRounds != 1 {
		t.Error("Expected 1 round to be available in the buffer")
	}
}

// Shows that GetCompletedBatch returns a completed batch when it's supposed
// to, and doesn't when it's not
func TestGetCompletedBatch(t *testing.T) {
	completedRounds := make(chan *mixmessages.Batch)

	// Should timeout
	batch, err := GetCompletedBatch(completedRounds, time.Second)
	if err == nil {
		t.Error("Should have gotten an error in the timeout case")
	}
	if batch != nil {
		t.Error("Should have gotten a nil batch in the timeout case")
	}

	// Should not timeout: writes to the completed rounds after an amount of
	// time
	time.AfterFunc(200*time.Millisecond, func() {
		completedRounds <- &mixmessages.Batch{}
	})
	batch, err = GetCompletedBatch(completedRounds, time.Second)
	if err != nil {
		t.Errorf("Got unexpected error on wait case: %v", err)
	}
	if batch == nil {
		t.Error("Expected a batch on wait case, got nil")
	}

	// Should not timeout: there's already a completed round on the channel
	go func() { completedRounds <- &mixmessages.Batch{} }()
	// This should allow the channel to be populated
    time.Sleep(10*time.Millisecond)
    batch, err = GetCompletedBatch(completedRounds, time.Second)
	if err != nil {
		t.Errorf("Got unexpected error on wait case: %v", err)
	}
	if batch == nil {
		t.Error("Expected a batch on wait case, got nil")
	}
}
