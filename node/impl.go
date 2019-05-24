////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package io impl.go implements server utility functions needed to work
// with the comms library
package node

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/server/io"
	"gitlab.com/elixxir/server/server"
	"time"
)

// NewImplementation creates a new implementation of the server.
// When a function is added to comms, you'll need to point to it here.
func NewImplementation(instance *server.Instance) *node.Implementation {

	impl := node.NewImplementation()

	//impl.Functions.RoundtripPing = func(*mixmessages.TimePing) {}
	//impl.Functions.GetServerMetrics = func(*mixmessages.ServerMetrics) {}
	//impl.Functions.CreateNewRound = func(message *mixmessages.RoundInfo) {}

	// impl.Functions.StartRealtime =

	impl.Functions.PostPhase = func(batch *mixmessages.Batch) {
		ReceivePostPhase(batch, instance)
	}

	impl.Functions.PostRoundPublicKey = func(pk *mixmessages.RoundPublicKey) {
		ReceivePostRoundPublicKey(instance, pk, impl)
	}

	// impl.Functions.RequestNonce =

	// impl.Functions.ConfirmRegistration =

	impl.Functions.GetRoundBufferInfo = func() (int, error) {
		return io.GetRoundBufferInfo(instance.GetCompletedPrecomps(), time.Second)
	}

	impl.Functions.GetCompletedBatch = func() (batch *mixmessages.Batch, e error) {
		return io.GetCompletedBatch(instance.GetCompletedBatchQueue(), time.Second)
	}
	//impl.Functions.PostRoundPublicKey =
	impl.Functions.FinishRealtime = func(message *mixmessages.RoundInfo) error {
		return io.FinishRealtime(instance.GetRoundManager(), message)
	}

	impl.Functions.RequestNonce = func(salt, Y, P, Q, G, hash, R, S []byte) ([]byte, error) {
		return io.RequestNonce(instance, salt, Y, P, Q, G, hash, R, S)
	}

	impl.Functions.ConfirmRegistration = func(hash, R, S []byte) ([]byte, []byte, []byte,
		[]byte, []byte, []byte, []byte, error) {
		return io.ConfirmRegistration(instance, hash, R, S)
	}
	impl.Functions.PostPrecompResult = func(roundID uint64, slots []*mixmessages.Slot) error {
		return ReceivePostPrecompResult(instance, roundID, slots)
	}
	impl.Functions.PostNewBatch = func(newBatch *mixmessages.Batch) error {
		return ReceivePostNewBatch(instance, newBatch)
	}
	return impl
}
