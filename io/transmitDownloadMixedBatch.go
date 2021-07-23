///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package io

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"strings"
)

// StartDownloadMixedBatch is the handler for the gateway -> server comms.
// Denotes that the gateway is ready to receive a completed batch, as it has
// up-to-date knowledge for the round sent. This endpoint sends a signal to a
// long-running thread to start streaming the completed batch
func StartDownloadMixedBatch(instance *internal.Instance,
	ready *pb.BatchReady, auth *connect.Auth) error {
	// Check that the sender is authenticated and is either their gateway or the temporary gateway
	if !auth.IsAuthenticated || !isValidID(auth.Sender.GetId(), &id.TempGateway, instance.GetGateway()) {
		jww.INFO.Printf("Failed auth gateway poll: %v", auth)
		return connect.AuthError(auth.Sender.GetId())
	}

	cr, ok := instance.GetCompletedBatch(id.Round(ready.RoundId))
	if !ok {
		return errors.Errorf("Could not find completed batch for round %d", ready.RoundId)
	}

	jww.INFO.Printf("Transmitting completed batch to gateway")
	err := instance.GetCompletedBatchSignal().Send(cr)
	if err != nil {
		return errors.Errorf("Could not prepare completed batch for sending: %v", err)
	}

	return nil
}

// TransmitStreamDownloadBatch is a server -> gateway send function. This
// streams the completed batch's slots over to the gateway. Called in a long running
// thread, initiated by a gateway's request (StartDownloadMixedBatch)
func TransmitStreamDownloadBatch(instance *internal.Instance,
	completedBatch *round.CompletedRound) error {

	slots := completedBatch.Round
	// Construct the header
	rid := uint64(completedBatch.RoundID)
	batchInfo := mixmessages.BatchInfo{Round: &mixmessages.RoundInfo{ID: rid}}
	gwHost, ok := instance.GetNetwork().GetHost(instance.GetGateway())
	if !ok {
		return errors.Errorf("Could not retrieve gateway host")
	}

	// Construct the completed batch
	batch := &mixmessages.CompletedBatch{
		RoundID: rid,
		Slots:   slots,
	}

	// Stream the slots
	err := instance.GetNetwork().DownloadMixedBatch(gwHost, batchInfo, batch)
	if err != nil && !strings.Contains(err.Error(), "Host address is blank") {
		return errors.Errorf("Could not stream completed batch to gateway for round %d: %v", completedBatch.RoundID, err)
	}

	return nil
}
