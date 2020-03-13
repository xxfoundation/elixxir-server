////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package io impl.go implements server utility functions needed to work
// with the comms library
package receivers

import (
	"gitlab.com/elixxir/comms/connect"
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

	impl.Functions.GetMeasure = func(message *mixmessages.RoundInfo,
		auth *connect.Auth) (*mixmessages.RoundMetrics, error) {
		return ReceiveGetMeasure(instance, message)
	}

	impl.Functions.PostPhase = func(batch *mixmessages.Batch, auth *connect.Auth) error {
		return ReceivePostPhase(batch, instance, auth)
	}

	impl.Functions.StreamPostPhase = func(streamServer mixmessages.Node_StreamPostPhaseServer, auth *connect.Auth) error {
		return ReceiveStreamPostPhase(streamServer, instance, auth)
	}

	impl.Functions.PostRoundPublicKey = func(pk *mixmessages.RoundPublicKey, auth *connect.Auth) error {
		return ReceivePostRoundPublicKey(instance, pk, auth)
	}

	impl.Functions.GetRoundBufferInfo = func(auth *connect.Auth) (int, error) {
		return 0, nil //io.GetRoundBufferInfo(instance.GetCompletedPrecomps(), time.Second)
	}

	impl.Functions.GetCompletedBatch = func(auth *connect.Auth) (batch *mixmessages.Batch, e error) {
		return nil, nil //io.GetCompletedBatch(instance, time.Second, auth)
	}

	impl.Functions.FinishRealtime = func(message *mixmessages.RoundInfo, auth *connect.Auth) error {
		return ReceiveFinishRealtime(instance, message, auth)
	}

	impl.Functions.RequestNonce = func(salt []byte, RSAPubKey string,
		DHPubKey, RSASignedByRegistration, DHSignedByClientRSA []byte, auth *connect.Auth) ([]byte, []byte, error) {
		return io.RequestNonce(instance, salt, RSAPubKey, DHPubKey,
			RSASignedByRegistration, DHSignedByClientRSA, auth)
	}

	impl.Functions.ConfirmRegistration = func(UserID, Signature []byte, auth *connect.Auth) ([]byte, error) {
		return io.ConfirmRegistration(instance, UserID, Signature, auth)
	}
	impl.Functions.PostPrecompResult = func(roundID uint64, slots []*mixmessages.Slot, auth *connect.Auth) error {
		return ReceivePostPrecompResult(instance, roundID, slots, auth)
	}
	impl.Functions.PostNewBatch = func(newBatch *mixmessages.Batch, auth *connect.Auth) error {
		return ReceivePostNewBatch(instance, newBatch, io.PostPhase, auth)
	}

	impl.Functions.SendRoundTripPing = func(ping *mixmessages.RoundTripPing, auth *connect.Auth) error {
		return ReceiveRoundTripPing(instance, ping)
	}

	impl.Functions.Poll = func(poll *mixmessages.ServerPoll, auth *connect.Auth) (*mixmessages.ServerPollResponse, error) {

		return nil, nil //receivers.ReceivePoll()
	}

	impl.Functions.AskOnline = func() error {
		for instance.Online == false {
			time.Sleep(250 * time.Millisecond)
		}
		return nil
	}
	return impl
}
