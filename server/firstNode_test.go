package server

import (
	"errors"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/id"
	"reflect"
	"testing"
	"time"
)

// tests that the proper queue is returned
func TestFirstNode_GetNewBatchQueue(t *testing.T) {
	fn := &firstNode{}
	fn.Initialize()

	if !reflect.DeepEqual(fn.newBatchQueue, fn.GetNewBatchQueue()) {
		t.Errorf("FirstNode.GetNewBatchQueue: returned queue not the same" +
			" as internal queue")
	}
}

// tests that the proper queue is returned
func TestFirstNode_GetCompletedPrecompQueue(t *testing.T) {
	fn := &firstNode{}
	fn.Initialize()

	if !reflect.DeepEqual(fn.readyRounds, fn.GetCompletedPrecomps()) {
		t.Errorf("FirstNode.GetCompletedPrecompQueue: returned queue not the same" +
			" as internal queue")
	}
}

var receivedRoundID id.Round

func mockTransmitter(n *node.NodeComms, c *circuit.Circuit, rID id.Round) error {
	receivedRoundID = rID
	return nil
}

func mockTransmitter_Error(n *node.NodeComms, c *circuit.Circuit, rID id.Round) error {
	receivedRoundID = rID
	return errors.New("test error")
}

// tests roundCreationRunner's happy path
func TestFirstNode_roundCreationRunner(t *testing.T) {

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("RoundCreationRunner: happy path test should not "+
				"error: %+v", r)
		}
	}()

	fn := firstNode{}
	fn.Initialize()

	fn.currentRoundID = 5

	fn.finishedRound <- fn.currentRoundID

	fn.roundCreationRunner(&Instance{}, 2*time.Millisecond,
		mockTransmitter, func(*Instance, id.Round) error { return nil })
}

// tests roundCreationRunner stops timeout when waiting a short
// period of time before sending the finished round
func TestFirstNode_roundCreationRunner_wait(t *testing.T) {

	fn := firstNode{}
	fn.Initialize()

	fn.currentRoundID = 5

	go func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("RoundCreationRunner: short wait test should not "+
					"error: %s", r)
			}
		}()

		fn.roundCreationRunner(&Instance{}, 2*time.Millisecond,
			mockTransmitter, func(*Instance, id.Round) error { return nil })
	}()

	time.After(1 * time.Millisecond)

	fn.finishedRound <- fn.currentRoundID
}

// tests roundCreationRunner times out appropriately
func TestFirstNode_roundCreationRunner_Timeout(t *testing.T) {

	defer func() {
		if r := recover(); r != nil {
			if r.(string) != "Round did not finish within timeout of 2ms" {
				t.Errorf("RoundCreationRunner: panic returned incorrect"+
					"for timeout error: %s", r)
			}
		}
	}()

	fn := firstNode{}
	fn.Initialize()

	fn.currentRoundID = 5

	//fn.finishedRound <- fn.currentRoundID

	fn.roundCreationRunner(&Instance{}, 2*time.Millisecond,
		mockTransmitter, func(*Instance, id.Round) error { return nil })

	t.Errorf("RoundCreationRunner: Timeout test did not timeout")
}

// tests roundCreationRunner panics when the network returns an error
func TestFirstNode_roundCreationRunner_NetworkError(t *testing.T) {

	defer func() {
		if r := recover(); r != nil {
			if r.(string) != "Round failed to create round remotely: test error" {
				t.Errorf("RoundCreationRunner: panic returned incorrect "+
					"error for network error: %s", r)
			}
		}
	}()

	fn := firstNode{}
	fn.Initialize()

	fn.currentRoundID = 5

	fn.finishedRound <- fn.currentRoundID

	fn.roundCreationRunner(&Instance{}, 2*time.Millisecond,
		mockTransmitter_Error, func(*Instance, id.Round) error { return nil })

	t.Errorf("RoundCreationRunner: Timeout test did not timeout")
}
