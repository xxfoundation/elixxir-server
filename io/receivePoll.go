////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package io

// receivePoll contains the handler for the gateway <-> server poll comm

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"strings"
)

// Handles incoming Poll gateway responses, compares our NDF with the existing ndf
func ReceivePoll(poll *mixmessages.ServerPoll, instance *internal.Instance,
	auth *connect.Auth) (*mixmessages.ServerPollResponse, error) {

	// Check that the sender is authenticated and is either their gateway or the temporary gateway
	if !auth.IsAuthenticated || !isValidID(auth.Sender.GetId(), &id.TempGateway, instance.GetGateway()) {
		jww.INFO.Printf("Failed auth gateway poll: %v", auth)
		return nil, connect.AuthError(auth.Sender.GetId())
	}

	res := mixmessages.ServerPollResponse{}

	// Put gateway address and version into gateway data in instance
	instance.UpsertGatewayData(poll.GatewayAddress, poll.GatewayVersion)

	// Asynchronously indicate that gateway has successfully contacted
	// its node
	instance.GetGatewayFirstContact().Send()

	earliestClientRoundId, earliestGwRoundId, earliestRoundTs, err := instance.GetEarliestRound()
	if err != nil {
		res.EarliestRoundErr = err.Error()
	} else {
		res.EarliestRoundTimestamp = earliestRoundTs
		res.EarliestGatewayRound = earliestGwRoundId
		res.EarliestClientRound = earliestClientRoundId
	}

	// Node is only ready for a response once it has polled permissioning
	if instance.IsReadyForGateway() {
		network := instance.GetNetworkStatus()

		//Compare partial NDF hash with instance and return the new one if they do not match
		isSame := network.GetPartialNdf().CompareHash(poll.GetPartial().Hash)
		if !isSame {
			res.PartialNDF = network.GetPartialNdf().GetPb()
		}

		// Get the request for a new batch que and store it into res
		res.BatchRequest, _ = instance.GetRequestNewBatchQueue().Receive()

		//get a completed batch round ID if it exists and pass it to the gateway
		rid, err := instance.GetCompletedBatchRID()
		if err != nil &&
			!strings.Contains(err.Error(), internal.NoCompletedBatch) {
			return nil, errors.Errorf("Unable to get from completedBatch: %v", err)
		}

		if rid != 0 {
			res.Batch = &mixmessages.BatchReady{RoundId: uint64(rid)}
		}

		//Compare Full NDF hash with instance and
		// return the new one if they do not match
		// otherwise return round updates
		isSame = network.GetFullNdf().CompareHash(
			poll.GetFull().Hash)
		if isSame {
			//Check if any updates where made and get them
			res.Updates = network.GetRoundUpdates(
				int(poll.LastUpdate))
		} else {
			res.FullNDF = network.GetFullNdf().GetPb()
		}

		// Populate the id field
		res.Id = instance.GetID().Bytes()

		// denote that gateway has received info,
		// only does something the first time
		instance.GetGatewayFirstPoll().Send()

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
