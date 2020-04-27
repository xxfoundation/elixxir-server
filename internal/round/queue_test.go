////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package round

import (
	"fmt"
	"gitlab.com/elixxir/comms/mixmessages"
	"reflect"
	"testing"
)

// Smoke test of new queue
func TestNewQueue(t *testing.T) {
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
func TestQueue_Send(t *testing.T) {
	ourNewQ := NewQueue()

	if len(ourNewQ) != 0 {
		t.Errorf("New Queue expected to be of length 0! Length is: %+v", len(ourNewQ))
	}

	err := ourNewQ.Send(&mixmessages.RoundInfo{})
	if err != nil {
		t.Errorf("Should be able to send when queue is empty: %+v."+
			"\nLength of queue: %+v", err, len(ourNewQ))
	}
}

// Error path: Attempt to send to an already full queue
func TestQueue_Send_Error(t *testing.T) {
	ourNewQ := NewQueue()

	// Send to queue once
	err := ourNewQ.Send(&mixmessages.RoundInfo{})
	if err != nil {
		t.Errorf("")
	}

	// Attempt to send again without emptying queue
	err = ourNewQ.Send(&mixmessages.RoundInfo{})
	if err != nil {
		return
	}

	t.Errorf("Expected error path, should not be able to send to a full queue")
}

// Happy path
func TestQueue_Receive(t *testing.T) {
	ourNewQ := NewQueue()

	ourRoundInfo := &mixmessages.RoundInfo{
		ID:       uint64(25),
		State:    uint32(52),
		Topology: []string{"te", "est", "testtest"},
	}

	// Send to queue
	err := ourNewQ.Send(ourRoundInfo)
	if err != nil {
		t.Errorf("Expected happy path, received error when sending! Err: %+v", err)
	}

	receivedRoundInfo, err := ourNewQ.Receive()

	if !reflect.DeepEqual(receivedRoundInfo, ourRoundInfo) {
		t.Errorf("Received round info does not match that put in!"+
			"Expected: %+v \n\t"+
			"Received: %+v", ourRoundInfo, receivedRoundInfo)
	}

	if err != nil {
		t.Errorf("Expected happy path, received error when receiving! Err: %+v", err)
	}
}

// Error path: Attempt to receive from an empty queue
func TestQueue_Receive_Error(t *testing.T) {
	ourNewQ := NewQueue()

	_, err := ourNewQ.Receive()

	if err != nil {
		return
	}

	t.Errorf("Expected error path, should not be able to receive from an empty queue!")
}
