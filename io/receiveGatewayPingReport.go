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
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"strings"
)

// ReceiveGatewayPingReport is a handler for ReportGatewayPings.
// Processes the results of their gateway pinging all gateways in the round's team.
// Pinging issues could be due to poor connectivity or a gateway not open for connections.
// If pinging issues are present and the round hasn't finished, the round is
// reported as a failure. Otherwise, this handler does nothing and
// the round continues as normal.
func ReceiveGatewayPingReport(report *pb.GatewayPingReport, auth *connect.Auth,
	instance *internal.Instance) error {

	// Check that the sender is authenticated and is
	//either their gateway or the temporary gateway
	if !auth.IsAuthenticated ||
		!isValidID(auth.Sender.GetId(), &id.TempGateway, instance.GetGateway()) {
		jww.INFO.Printf("Failed auth gateway poll: %v", auth)
		return connect.AuthError(auth.Sender.GetId())
	}

	// Get round from consensus
	roundID := id.Round(report.RoundId)
	ri, err := instance.GetConsensus().GetRound(roundID)
	if err != nil {
		return errors.Errorf("Could not get round %d from consensus", roundID)
	}

	roundState := states.Round(ri.GetState())

	// Check if the round has already finished
	// (either due to completion or failure)
	if roundState == states.FAILED ||
		roundState == states.COMPLETED {
		return nil
	}

	// Initiate round error if there are un-pingable gateays
	if len(report.FailedGateways) != 0 {
		gwIdList, err := id.NewIDListFromBytes(report.FailedGateways)
		if err != nil {
			jww.WARN.Printf("ReceiveGatewayPingReport: " +
				"Could not parse list of un-pingable gateways sent by our gateway.")
			roundErr := errors.Errorf("Round %d failed due to team node(s) having "+
				"un-contactable gateway(s). Could not construct list of nodes.", roundID)
			instance.ReportRoundFailure(roundErr, instance.GetID(), roundID)
			return nil
		}

		// Collect list of nodeId's that had un-pingable gateways
		nodeList := make([]string, 0)
		for _, gwId := range gwIdList {
			nodeId := gwId.DeepCopy()
			nodeId.SetType(id.Node)
			nodeList = append(nodeList, nodeId.String())
		}

		// Reformat node list to a human readProblematic node IDs as followsable format
		nodeListErr := strings.Join(nodeList, ", ")
		roundErr := errors.Errorf("Round %d failed due to team node(s) having "+
			"un-contactable gateway(s). Problematic node IDs as follows: %v", roundID, nodeListErr)
		instance.ReportRoundFailure(roundErr, instance.GetID(), roundID)
	}

	return nil
}
