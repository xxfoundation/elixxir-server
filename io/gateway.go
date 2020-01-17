package io

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/server/server"
	"time"
)

// gateway.go is for gateway<->node comms

func GetRoundBufferInfo(roundBuffer *server.PrecompBuffer,
	timeout time.Duration) (int, error) {
	// Verify completed precomputations
	if roundBuffer == nil {
		time.Sleep(timeout)
		return 0, nil
	}
	numRounds := len(roundBuffer.CompletedPrecomputations)
	if numRounds != 0 {
		// There are rounds ready, so return
		// Note: This should be considered an edge case
		return len(roundBuffer.CompletedPrecomputations), nil
	} else {
		// Wait for a round to be pushed
		select {
		case <-roundBuffer.PushSignal:
			// Succeed
			return len(roundBuffer.CompletedPrecomputations), nil
		case <-time.After(timeout):
			// Timeout and fail
			return len(roundBuffer.CompletedPrecomputations), nil
		}
	}
}

// Returns a completed batch, or waits for a small amount of time for one to
// materialize if there isn't one ready
func GetCompletedBatch(instance *server.Instance,
	timeout time.Duration, auth *connect.Auth) (*mixmessages.Batch, error) {

	// Check that authentication is good and the sender is our gateway, otherwise error
	if !auth.IsAuthenticated || auth.Sender.GetId() != instance.GetID().NewGateway().String() {
		jww.INFO.Printf("[%s]: GetCompletedBatch failed auth (sender ID: %s, auth: %v)",
			instance, auth.Sender.GetId(), auth.IsAuthenticated)
		return nil, connect.AuthError(auth.Sender.GetId())
	}

	var roundQueue *server.CompletedRound
	select {
	case roundQueue = <-instance.GetCompletedBatchQueue():
	case <-time.After(timeout):
		return &mixmessages.Batch{}, nil
	}

	//build the batch
	batch := &mixmessages.Batch{
		Round: &mixmessages.RoundInfo{ID: uint64(roundQueue.RoundID)},
	}

	var slots []*mixmessages.Slot

	for chunk, ok := roundQueue.GetChunk(); ok; chunk, ok = roundQueue.GetChunk() {
		for i := chunk.Begin(); i < chunk.End(); i++ {
			slots = append(slots, roundQueue.GetMessage(i))
		}
	}

	batch.Slots = slots

	return batch, nil

}
