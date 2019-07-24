////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package io metrics.go handles the endpoints and helper functions for
// receiving and sending the metrics message between cMix nodes.

package io

import (
	"encoding/json"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/id"
)

// Get metrics for all nodes in the topology, returning a JSON map of server address to metrics
func TransmitGetMeasure(node *node.NodeComms, topology *circuit.Circuit, roundID id.Round) string {
	serverMetrics := map[string]string{}

	// Contact all visible servers and get metrics
	for i := 0; i < topology.Len(); i++ {
		server := topology.GetNodeAtIndex(i)

		metric, err := node.SendGetMeasure(server, &pb.RoundInfo{
			ID: uint64(roundID),
		})

		if err != nil {
			jww.ERROR.Printf("Could not contact cMix server %s (%d/%d)...",
				server, i+1, topology.Len())
			serverMetrics[server.String()] = fmt.Sprintf("Error: could not contact server %s", server.String())
			continue
		}

		serverMetrics[server.String()] = metric.RoundMetricJSON
	}

	// Marshal server metrics into a JSON
	ret, err := json.Marshal(serverMetrics)

	if err != nil {
		jww.ERROR.Printf("Could not form final JSON")
	}
	return string(ret)
}
