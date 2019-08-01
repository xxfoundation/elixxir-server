package io

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/services"
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
	rbi, _ := GetRoundBufferInfo(c, 2*time.Millisecond)
	if rbi != 0 {
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

	completedRoundQueue := make(chan *server.CompletedRound)

	doneChan := make(chan struct{})

	var batch *mixmessages.Batch

	// Should timeout
	go func() {
		batch, _ = GetCompletedBatch(completedRoundQueue, 40*time.Millisecond)

		doneChan <- struct{}{}

	}()

	<-doneChan

	if len(batch.Slots) != 0 {
		t.Error("Should have gotten an error in the timeout case")
	}
}

func TestGetCompletedBatch_ShortWait(t *testing.T) {
	// Not a timeout: There's an actual completed batch available in the
	// channel after a certain period of time
	completedRoundQueue := make(chan *server.CompletedRound, 1)

	// Should not timeout: writes to the completed rounds after an amount of
	// time

	var batch *mixmessages.Batch
	var err error

	doneChan := make(chan struct{})

	complete := &server.CompletedRound{
		RoundID:    42, //meaning of life
		Receiver:   make(chan services.Chunk),
		GetMessage: func(uint32) *mixmessages.Slot { return nil },
	}

	go func() {
		batch, err = GetCompletedBatch(completedRoundQueue, 20*time.Millisecond)
		doneChan <- struct{}{}
	}()

	time.After(5 * time.Millisecond)

	completedRoundQueue <- complete

	complete.Receiver <- services.NewChunk(0, 3)

	close(complete.Receiver)

	<-doneChan

	if err != nil {
		t.Errorf("Got unexpected error on wait case: %v", err)
	}
	if batch == nil {
		t.Error("Expected a batch on wait case, got nil")
	}
}

func TestGetCompletedBatch_BatchReady(t *testing.T) {
	// If there's already a completed batch, the comm should get it immediately
	completedRoundQueue := make(chan *server.CompletedRound, 1)

	// Should not timeout: writes to the completed rounds after an amount of
	// time

	var batch *mixmessages.Batch
	var err error

	doneChan := make(chan struct{})

	complete := &server.CompletedRound{
		RoundID:    42, //meaning of life
		Receiver:   make(chan services.Chunk, 1),
		GetMessage: func(uint32) *mixmessages.Slot { return nil },
	}

	completedRoundQueue <- complete

	complete.Receiver <- services.NewChunk(0, 3)

	go func() {
		batch, err = GetCompletedBatch(completedRoundQueue, 20*time.Millisecond)
		doneChan <- struct{}{}
	}()

	close(complete.Receiver)

	<-doneChan

	if err != nil {
		t.Errorf("Got unexpected error on wait case: %v", err)
	}
	if batch == nil {
		t.Error("Expected a batch on wait case, got nil")
	}
}
