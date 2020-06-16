///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package round

import (
	"fmt"
	"gitlab.com/elixxir/comms/mixmessages"
	"testing"
)

// Smoke test of new queue
func TestNewCompletedQueue(t *testing.T) {
	ourNewQ := NewQueue()

	if len(ourNewQ) != 0 {
		t.Errorf("New Queue expected to be of length 0! Length is: %+v", len(ourNewQ))
	}

	// Test
	ourNewQ <- &mixmessages.RoundInfo{}

	if len(ourNewQ) != 1 {
		t.Errorf("Queue expected to be of length 1! Length is: %+v", len(ourNewQ))
	}

	select {
	case ourNewQ <- &mixmessages.RoundInfo{}: // Put 2 in the channel unless it is full
		t.Errorf("Channel should be full, should not be able to put additional value into it")
	default:
		fmt.Println("Channel full. Discarding value")
	}

}

// Happy path
func TestCompletedQueue_Send(t *testing.T) {
	ourNewQ := NewCompletedQueue()

	if len(ourNewQ) != 0 {
		t.Errorf("New Queue expected to be of length 0! Length is: %+v", len(ourNewQ))
	}

	cr := CompletedRound{RoundID: 1, Round: nil}
	err := ourNewQ.Send(&cr)
	if err != nil {
		t.Errorf("Should be able to send when queue is empty: %+v."+
			"\nLength of queue: %+v", err, len(ourNewQ))
	}
}

// Error path: Attempt to send to an already full queue
func TestCompletedQueue_Send_Send_Error(t *testing.T) {
	ourNewQ := NewCompletedQueue()
	cr := CompletedRound{RoundID: 1, Round: nil}
	// Send to queue once
	err := ourNewQ.Send(&cr)
	if err != nil {
		t.Errorf("")
	}

	// Attempt to send again without emptying queue
	for i := 0; i < maxCompletedBatches; i++ {
		_ = ourNewQ.Send(&cr)
	}
	err = ourNewQ.Send(&cr)
	if err != nil {
		return
	}

	t.Errorf("Expected error path, should not be able to send to a full queue")
}

// Happy path
func TestCompletedQueue_Receive(t *testing.T) {
	ourNewQ := NewCompletedQueue()

	cr := CompletedRound{RoundID: 23, Round: nil}
	// Send to queue
	err := ourNewQ.Send(&cr)
	if err != nil {
		t.Errorf("Expected happy path, received error when sending! Err: %+v", err)
	}

	receivedRoundInfo, err := ourNewQ.Receive()
	if err != nil {
		t.Errorf("Expected happy path, received error when receiving! Err: %+v", err)
	}

	if receivedRoundInfo.RoundID != 23 {
		t.Logf("Recieved unexpected round id")
		t.Fail()
	}

}

// Error path: Attempt to receive from an empty queue
func TestCompleteQueue_Receive_Error(t *testing.T) {
	ourNewQ := NewCompletedQueue()

	_, err := ourNewQ.Receive()

	if err != nil {
		return
	}

	t.Errorf("Expected error path, should not be able to receive from an empty queue!")
}
