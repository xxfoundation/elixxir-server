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

	jww.TRACE.Printf("Gateway Info: %s, %s", poll.GatewayAddress,
		poll.GatewayVersion)

	// fixme: this is a hack. due to the interconnected web of server <-> permissioning polling
	//  and GW <-> server polling, we reach a deadlock state
	//  To elaborate: GW only sets its address AFTER polling server for an NDF and
	//   checking Connectivity w/ permissioning (GW needs to poll server to contact permissioning in the first place)
	//   so it sends an empty address to server until this is set
	//  However server needs GW's address BEFORE registering as it needs to provide valid addresses for the NDF
	//  Server does not provide an NDF to GW until it (server) has registered and polled permissioning
	//  Which means GW cannot get a response when it polls server and cannot move forward to setting its own address
	//  Therefore GW doesn't ever get it's address set, and server crashes when trying to parse an empty GW address
	// fixme: Suggested solution: Generate a comm where server gets GW's address, and place that comm in RegisterNode
	// Cannot upsert gateway data unless valid address is sent through the wire
	if poll.GatewayAddress == "" {
		poll.GatewayAddress = "1.2.3.4:11420"
	}
	// Form gateway address and put it into gateway data in instance
	instance.UpsertGatewayData(poll.GatewayAddress, poll.GatewayVersion)

	// Asynchronously indicate that gateway has successfully contacted
	// its node
	instance.GetGatewayFirstContact().Send()


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
		if err != nil && !strings.Contains(err.Error(), "Did not receive a completed round") {
			return nil, errors.Errorf("Unable to receive from CompletedBatchQueue: %+v", err)
		}

		if cr != nil {
			res.Slots = cr.Round
		}

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
