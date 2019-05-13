package io

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/round"
	"testing"
	"time"
)

func TestGetRoundBufferInfo_RoundsInBuffer(t *testing.T) {
	// This is actually an edge case: the number of available precomps is
	// greater than zero. This should only happen in production if the
	// communication between the gateway and the node breaks down.
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
}

func TestGetRoundBufferInfo_Timeout(t *testing.T) {
	// More than timeout case: length is zero and stays there
	c := &server.PrecompBuffer{
		CompletedPrecomputations: make(chan *round.Round, 1),
		PushSignal:               make(chan struct{}),
	}
	_, err := GetRoundBufferInfo(c, 200*time.Millisecond)
	if err == nil {
		t.Error("Round buffer info timeout should have resulted in an error")
	}
}

func TestGetRoundBufferInfo_LessThanTimeout(t *testing.T) {
	// Tests less than timeout case: length that's zero, then one,
	// should result in a length of one
	c := &server.PrecompBuffer{
		CompletedPrecomputations: make(chan *round.Round, 1),
		PushSignal:               make(chan struct{}),
	}
	before := time.Now()
	time.AfterFunc(200*time.Millisecond, func() {
		c.Push(nil)
	})
	availableRounds, err := GetRoundBufferInfo(c, time.Second)
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

func TestGetCompletedBatch_Timeout(t *testing.T) {
	// Timeout case: There are no batches completed
	completedRounds := make(chan *mixmessages.Batch)

	// Should timeout
	batch, err := GetCompletedBatch(completedRounds, time.Second)
	if err == nil {
		t.Error("Should have gotten an error in the timeout case")
	}
	if batch != nil {
		t.Error("Should have gotten a nil batch in the timeout case")
	}
}

func TestGetCompletedBatch_(t *testing.T) {
	// Not a timeout: There's an actual completed batch available in the
	// channel after a certain period of time
	completedRounds := make(chan *mixmessages.Batch)

	// Should not timeout: writes to the completed rounds after an amount of
	// time
	time.AfterFunc(200*time.Millisecond, func() {
		completedRounds <- &mixmessages.Batch{}
	})
	batch, err := GetCompletedBatch(completedRounds, time.Second)
	if err != nil {
		t.Errorf("Got unexpected error on wait case: %v", err)
	}
	if batch == nil {
		t.Error("Expected a batch on wait case, got nil")
	}
}

func TestGetCompletedBatch3(t *testing.T) {
	// If there's already a completed batch, the comm should get it immediately
	completedRounds := make(chan *mixmessages.Batch)
	// Should not timeout: there's already a completed round on the channel
	go func() { completedRounds <- &mixmessages.Batch{} }()
	// This should allow the channel to be populated
	time.Sleep(10 * time.Millisecond)
	batch, err := GetCompletedBatch(completedRounds, time.Second)
	if err != nil {
		t.Errorf("Got unexpected error on wait case: %v", err)
	}
	if batch == nil {
		t.Error("Expected a batch on wait case, got nil")
	}
}
