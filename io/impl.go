///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Package io impl.go implements server utility functions needed to work
// with the comms library
package io

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/interconnect"
	"gitlab.com/xx_network/primitives/ndf"
	"time"
)

// NewImplementation creates a new implementation of the server.
// When a function is added to comms, you'll need to point to it here.
func NewImplementation(instance *internal.Instance) *node.Implementation {

	impl := node.NewImplementation()

	impl.Functions.GetMeasure = func(message *mixmessages.RoundInfo,
		auth *connect.Auth) (*mixmessages.RoundMetrics, error) {
		metrics, err := ReceiveGetMeasure(instance, message, auth)
		if err != nil {
			jww.ERROR.Printf("GetMeasure error: %+v, %+v", auth, err)
		}
		return metrics, err

	}

	impl.Functions.DownloadMixedBatch = func(stream pb.Node_DownloadMixedBatchServer, batchInfo *pb.BatchReady, auth *connect.Auth) error {
		return DownloadMixedBatch(instance, batchInfo, stream, auth)
	}

	impl.Functions.GetNdf = func() (*interconnect.NDF, error) {
		response, err := GetNdf(instance)
		if err != nil {
			jww.ERROR.Printf("GetNdf error: %v", err)
		}

		return &interconnect.NDF{
			Ndf: response,
		}, err
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

	impl.Functions.FinishRealtime = func(message *mixmessages.RoundInfo,
		streamServer pb.Node_FinishRealtimeServer, auth *connect.Auth) error {
		err := ReceiveFinishRealtime(instance, message, streamServer, auth)
		if err != nil {
			jww.ERROR.Printf("ReceiveFinishRealtime error: %+v, %+v", auth, err)
		}
		return err

	}

	impl.Functions.RequestNonce = func(nonceRequest *pb.NonceRequest, auth *connect.Auth) (*pb.Nonce, error) {
		nonce, err := RequestNonce(instance, nonceRequest, auth)
		if err != nil {
			jww.ERROR.Printf("RequestNonce error: %+v, %+v", auth, err)
		}
		return nonce, err
	}

	impl.Functions.ConfirmRegistration = func(confirmationRequest *pb.RequestRegistrationConfirmation,
		auth *connect.Auth) (*pb.RegistrationConfirmation, error) {
		response, err := ConfirmRegistration(instance, confirmationRequest, auth)
		if err != nil {
			jww.ERROR.Printf("ConfirmRegistration failed auth: %+v, %+v", auth, err)
		}
		return response, err
	}
	impl.Functions.PostPrecompResult = func(roundID uint64, slots []*mixmessages.Slot, auth *connect.Auth) error {
		err := ReceivePostPrecompResult(instance, roundID, slots, auth)
		if err != nil {
			jww.ERROR.Printf("ReceivePostPrecompResult error: %+v, %+v", auth, err)
		}
		return err
	}

	impl.Functions.UploadUnmixedBatch = func(stream pb.Node_UploadUnmixedBatchServer, auth *connect.Auth) error {
		err := ReceiveUploadUnmixedBatchStream(instance, stream, PostPhase, auth)
		if err != nil {
			jww.ERROR.Printf("ReceiveUploadUnmixedBatchStream error: %+v, %+v", auth, err)
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
		response, err := ReceivePoll(poll, instance, auth)
		if err != nil && err.Error() != ndf.NO_NDF {
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

	impl.Functions.RoundError = func(error *mixmessages.RoundError, auth *connect.Auth) error {
		err := ReceiveRoundError(error, auth, instance)
		if err != nil {
			jww.ERROR.Printf("[%v] ReceiveRoundError error: %v", instance, err.Error())
			return err
		}
		return nil
	}

	impl.Functions.GetPermissioningAddress = func() (string, error) {
		address, err := ReceivePermissioningAddressPing(instance)
		if err != nil {
			jww.ERROR.Printf("Failed to receive ping from gateway requesting "+
				"permissioning address: %+v", err)
			return "", err
		}
		return address, nil
	}

	impl.Functions.StartSharePhase = func(ri *mixmessages.RoundInfo, auth *connect.Auth) error {
		return ReceiveStartSharePhase(ri, auth, instance)
	}

	impl.Functions.SharePhaseRound = func(sharedPiece *pb.SharePiece, auth *connect.Auth) error {
		return ReceiveSharePhasePiece(sharedPiece, auth, instance)
	}

	impl.Functions.ShareFinalKey = func(sharedPiece *pb.SharePiece, auth *connect.Auth) error {
		return ReceiveFinalKey(sharedPiece, auth, instance)
	}

	return impl
}
