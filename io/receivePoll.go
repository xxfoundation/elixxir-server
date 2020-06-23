///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package io

// receivePoll contains the handler for the gateway <-> server poll comm

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
	"gitlab.com/elixxir/server/internal"
	"strings"
)

// Handles incoming Poll gateway responses, compares our NDF with the existing ndf
func ReceivePoll(poll *mixmessages.ServerPoll, instance *internal.Instance, gatewayAddress string,
	auth *connect.Auth) (*mixmessages.ServerPollResponse, error) {

	// Check that the sender is authenticated and is either their gateway or the temporary gateway
	if !auth.IsAuthenticated || (!isValidID(auth.Sender.GetId(), &id.TempGateway, instance.GetGatewayID())) {
		jww.TRACE.Printf("Failed auth object: %v", auth)
		return nil, connect.AuthError(auth.Sender.GetId())
	}

	res := mixmessages.ServerPollResponse{}

	jww.TRACE.Printf("Gateway Info: %s, %s", gatewayAddress,
		poll.GatewayVersion)

	// Form gateway address and put it into gateway data in instance
	instance.UpsertGatewayData(gatewayAddress, poll.GatewayVersion)

	// Node is only ready for a response once it has polled permissioning
	if instance.IsReadyForGateway() {
		network := instance.GetConsensus()

		//Compare partial NDF hash with instance and return the new one if they do not match
		isSame := network.GetPartialNdf().CompareHash(poll.GetPartial().Hash)
		if !isSame {
			res.PartialNDF = network.GetPartialNdf().GetPb()
		}

		//Compare Full NDF hash with instance and return the new one if they do not match
		isSame = network.GetFullNdf().CompareHash(poll.GetFull().Hash)
		if !isSame {
			res.FullNDF = network.GetFullNdf().GetPb()
		}

		// Populate the id field
		res.Id = instance.GetID().Bytes()

		//Check if any updates where made and get them
		res.Updates = network.GetRoundUpdates(int(poll.LastUpdate))

		// Get the request for a new batch que and store it into res
		res.BatchRequest, _ = instance.GetRequestNewBatchQueue().Receive()

		//get a completed batch if it exists and pass it to the gateway
		cr, err := instance.GetCompletedBatchQueue().Receive()
		if err != nil && !strings.Contains(err.Error(), "Did not recieve a completed round") {
			return nil, errors.Errorf("Unable to receive from CompletedBatchQueue: %+v", err)
		}

		if cr != nil {
			res.Slots = cr.Round
		}

		return &res, nil
	}

	// If node has not gotten a response from permissioning, return an empty message
	return &res, errors.New(ndf.NO_NDF)
}

// checks the sender against all passed in IDs, returning true if any match
// and skipping any that are nil
func isValidID(sender *id.ID, valid ...*id.ID) bool {
	for _, validID := range valid {
		if validID == nil {
			continue
		}
		if sender.Cmp(validID) {
			return true
		}
	}
	return false
}
