////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
)

// DownloadMixedBatch is the handler for the gateway -> server comms.
// Denotes that the gateway is ready to receive a completed batch, as it has
// up-to-date knowledge for the round sent. This endpoint streams the batch back to the gateway
func DownloadMixedBatch(instance *internal.Instance,
	ready *pb.BatchReady, stream pb.Node_DownloadMixedBatchServer, auth *connect.Auth) error {
	// Check that the sender is authenticated and is either their gateway or the temporary gateway
	if !auth.IsAuthenticated || !isValidID(auth.Sender.GetId(), &id.TempGateway, instance.GetGateway()) {
		jww.INFO.Printf("Failed auth gateway poll: %v", auth)
		return connect.AuthError(auth.Sender.GetId())
	}

	cr, ok := instance.GetCompletedBatch(id.Round(ready.RoundId))
	if !ok {
		return errors.Errorf("Could not find completed batch for round %d", ready.RoundId)
	}

	jww.INFO.Printf("Sending mixed batch for round %d to gateway", cr.RoundID)

	for i, slot := range cr.Round {
		if err := stream.Send(slot); err != nil {
			return errors.Errorf("Failed to send slot %d/%d for round %d",
				i, len(cr.Round), cr.RoundID)
		}
	}

	defer stream.Context().Done()

	return nil
}
