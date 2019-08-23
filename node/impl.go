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

	impl.Functions.CreateNewRound = func(message *mixmessages.RoundInfo) error {
		return ReceiveCreateNewRound(instance, message)
	}

	impl.Functions.GetMeasure = func(message *mixmessages.RoundInfo) (*mixmessages.RoundMetrics, error) {
		return ReceiveGetMeasure(instance, message)
	}

	impl.Functions.PostPhase = func(batch *mixmessages.Batch) {
		ReceivePostPhase(batch, instance)
	}

	impl.Functions.StreamPostPhase = func(streamServer mixmessages.Node_StreamPostPhaseServer) error {
		return ReceiveStreamPostPhase(streamServer, instance)
	}

	impl.Functions.PostRoundPublicKey = func(pk *mixmessages.RoundPublicKey) {
		ReceivePostRoundPublicKey(instance, pk)
	}

	impl.Functions.GetRoundBufferInfo = func() (int, error) {
		return io.GetRoundBufferInfo(instance.GetCompletedPrecomps(), time.Second)
	}

	impl.Functions.GetCompletedBatch = func() (batch *mixmessages.Batch, e error) {
		return io.GetCompletedBatch(instance.GetCompletedBatchQueue(), time.Second)
	}

	// Receive finish realtime should gather metrics if first node
	if instance.IsFirstNode() {
		impl.Functions.FinishRealtime = func(message *mixmessages.RoundInfo) error {
			return ReceiveFinishRealtime(instance, message)
		}
	} else {
		impl.Functions.FinishRealtime = func(message *mixmessages.RoundInfo) error {
			return ReceiveFinishRealtime(instance, message)
		}
	}

	impl.Functions.RequestNonce = func(salt []byte, RSAPubKey string,
		DHPubKey, RSASignedByRegistration, DHSignedByClientRSA []byte) ([]byte, []byte, error) {
		return io.RequestNonce(instance, salt, RSAPubKey, DHPubKey,
			RSASignedByRegistration, DHSignedByClientRSA)
	}

	impl.Functions.ConfirmRegistration = func(UserID, Signature []byte) ([]byte, error) {
		return io.ConfirmRegistration(instance, UserID, Signature)
	}
	impl.Functions.PostPrecompResult = func(roundID uint64, slots []*mixmessages.Slot) error {
		return ReceivePostPrecompResult(instance, roundID, slots)
	}
	impl.Functions.PostNewBatch = func(newBatch *mixmessages.Batch) error {
		return ReceivePostNewBatch(instance, newBatch)
	}

	// NOTE: AskOnline is notably absent here, despite having a transmitter.
	//       Until server start up is complicated enough to have state we
	//       need to check before it can process messages, we leave
	//       the simple ping response in comms lib for processing the RPC.
	return impl
}
