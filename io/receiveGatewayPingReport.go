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
	"gitlab.com/elixxir/primitives/current"
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

	// Check if the round is our latest round. If it is not, do nothing
	roundID := id.Round(report.RoundId)
	latestRound := instance.GetRoundManager().GetLatestRound()
	if latestRound != roundID {
		return nil
	}

	// Check if we are past precomputing stage. If so,
	// it's too late to fail, so we return
	if instance.GetStateMachine().Get() != current.PRECOMPUTING {
		nodeList, err := constructPrintableNodeIds(report)
		if err != nil {
			jww.WARN.Printf("ReceiveGatewayPingReport: %v", err)
			return nil

		}

		if len(report.FailedGateways) != 0 {
			jww.WARN.Printf("ReceiveGatewayPingReport: Round %d has "+
				"progressed too far "+
				"to handle a non-pingable gateway error."+
				"Problematic node ID(s) as follows: [%v]", nodeList)
		}
		return nil
	}

	// Initiate round error if there are un-pingable gateways
	if len(report.FailedGateways) != 0 {
		nodeList, err := constructPrintableNodeIds(report)
		if err != nil {
			jww.WARN.Printf("ReceiveGatewayPingReport: %v", err)
			roundErr := errors.Errorf("ReceiveGatewayPingReport: "+
				"Round %d failed due to team node(s) having "+
				"un-contactable gateway(s). Could not construct list of nodes.",
				roundID)
			instance.ReportRoundFailure(roundErr, instance.GetID(), roundID)
			return nil

		}
		roundErr := errors.Errorf("ReceiveGatewayPingReport: "+
			"Round %d failed due to team node(s) having "+
			"un-contactable gateway(s). Problematic node ID(s) as follows: [%v]",
			roundID, nodeList)
		instance.ReportRoundFailure(roundErr, instance.GetID(), roundID)
	}

	return nil
}

// Helper function which gets the node Ids of the un-contactable gateways,
// returning a human readable list for use in printing to a log
func constructPrintableNodeIds(report *pb.GatewayPingReport) (string, error) {
	// Parse the gateway Id list
	gwIdList, err := id.NewIDListFromBytes(report.FailedGateways)
	if err != nil {
		return "", errors.New("Could not parse list of un-pingable gateways sent by our gateway.")
	}

	// Collect list of nodeId's that had un-pingable gateways
	nodeList := make([]string, 0)
	for _, gwId := range gwIdList {
		nodeId := gwId.DeepCopy()
		nodeId.SetType(id.Node)
		nodeList = append(nodeList, nodeId.String())
	}

	// Reformat node list to a human readable format
	return strings.Join(nodeList, ", "), nil

}
