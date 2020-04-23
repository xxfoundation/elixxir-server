////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package io impl.go implements server utility functions needed to work
// with the comms library
package receivers

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/server/internals"
	"gitlab.com/elixxir/server/io/transmitters"
	"time"
)

// NewImplementation creates a new implementation of the server.
// When a function is added to comms, you'll need to point to it here.
func NewImplementation(instance *server.Instance) *node.Implementation {

	impl := node.NewImplementation()

	impl.Functions.GetMeasure = func(message *mixmessages.RoundInfo,
		auth *connect.Auth) (*mixmessages.RoundMetrics, error) {
		metrics, err := ReceiveGetMeasure(instance, message)
		if err != nil {
			jww.ERROR.Printf("GetMeasure error: %+v, %+v", auth, err)
		}
		return metrics, err

	}

	impl.Functions.PostPhase = func(batch *mixmessages.Batch, auth *connect.Auth) error {
		err := ReceivePostPhase(batch, instance, auth)
		if err != nil {
			jww.ERROR.Printf("ReceivePostPhase error: %+v, %+v", auth, err)
		}
		return err
	}

	impl.Functions.StreamPostPhase = func(streamServer mixmessages.Node_StreamPostPhaseServer, auth *connect.Auth) error {
		err := ReceiveStreamPostPhase(streamServer, instance, auth)
		if err != nil {
			jww.ERROR.Printf("StreamPostPhase error: %+v, %+v", auth, err)
		}
		return err
	}

	impl.Functions.PostRoundPublicKey = func(pk *mixmessages.RoundPublicKey, auth *connect.Auth) error {
		err := ReceivePostRoundPublicKey(instance, pk, auth)
		if err != nil {
			jww.ERROR.Printf("ReceivePostRoundPublicKey error: %+v, %+v", auth,
				err)
		}
		return err
	}

	impl.Functions.FinishRealtime = func(message *mixmessages.RoundInfo, auth *connect.Auth) error {
		err := ReceiveFinishRealtime(instance, message, auth)
		if err != nil {
			jww.ERROR.Printf("ReceiveFinishRealtime error: %+v, %+v", auth, err)
		}
		return err

	}

	impl.Functions.RequestNonce = func(salt []byte, RSAPubKey string,
		DHPubKey, RSASignedByRegistration, DHSignedByClientRSA []byte, auth *connect.Auth) ([]byte, []byte, error) {
		nonce, dhPub, err := io.RequestNonce(instance, salt, RSAPubKey, DHPubKey,
			RSASignedByRegistration, DHSignedByClientRSA, auth)
		if err != nil {
			jww.ERROR.Printf("RequestNonce error: %+v, %+v", auth, err)
		}
		return nonce, dhPub, err
	}

	impl.Functions.ConfirmRegistration = func(UserID, Signature []byte, auth *connect.Auth) ([]byte, error) {
		bytes, err := io.ConfirmRegistration(instance, UserID, Signature, auth)
		if err != nil {
			jww.ERROR.Printf("ConfirmRegistration failed auth: %+v, %+v", auth, err)
		}
		return bytes, err
	}
	impl.Functions.PostPrecompResult = func(roundID uint64, slots []*mixmessages.Slot, auth *connect.Auth) error {
		err := ReceivePostPrecompResult(instance, roundID, slots, auth)
		if err != nil {
			jww.ERROR.Printf("ReceivePostPrecompResult error: %+v, %+v", auth, err)
		}
		return err
	}

	impl.Functions.PostNewBatch = func(newBatch *mixmessages.Batch, auth *connect.Auth) error {
		err := ReceivePostNewBatch(instance, newBatch, io.PostPhase, auth)
		if err != nil {
			jww.ERROR.Printf("ReceivePostNewBatch error: %+v, %+v", auth, err)
		}
		return err
	}

	impl.Functions.SendRoundTripPing = func(ping *mixmessages.RoundTripPing, auth *connect.Auth) error {
		err := ReceiveRoundTripPing(instance, ping)
		if err != nil {
			jww.ERROR.Printf("SendRoundTripPing error: %+v, %+v", auth, err)
		}
		return err
	}

	impl.Functions.Poll = func(poll *mixmessages.ServerPoll, auth *connect.Auth) (*mixmessages.ServerPollResponse, error) {
		response, err := ReceivePoll(poll, instance)
		if err != nil {
			jww.ERROR.Printf("Poll error: %v, %+v", auth, err)
		}
		return response, err
	}

	impl.Functions.AskOnline = func() error {
		for instance.Online == false {
			time.Sleep(250 * time.Millisecond)
		}
		return nil
	}
	return impl
}
