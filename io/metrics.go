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
	"errors"
	"fmt"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server/measure"
	"strings"
)

const errorDelimiter = "; "

// Get metrics for all nodes in the topology, returning a JSON map of server
// address to metrics.
func TransmitGetMeasure(network *node.Comms, topology *circuit.Circuit, roundID id.Round) ([]measure.RoundMetrics, error) {

	// Stores errors for each SendGetMeasure() call to be concatenated on return
	var errs []string

	// Contact all visible servers and get metrics
	roundMetrics := make([]measure.RoundMetrics, topology.Len())

	// Loop through all the nodes
	for i := 0; i < topology.Len(); i++ {
		// Pull the particular server host object from the commManager
		currentNodeID := topology.GetNodeAtIndex(i).String()
		cuurentNode, ok := network.Manager.GetHost(currentNodeID)
		if !ok {
			errMsg := fmt.Sprintf("Could not find cMix server %s (%d/%d)  in comm manager",
				currentNodeID, i+1, topology.Len())
			errs = append(errs, errMsg)
		}
		roundMetric := measure.RoundMetrics{}

		metric, err := network.SendGetMeasure(cuurentNode, &pb.RoundInfo{
			ID: uint64(roundID),
		})

		// If there was an error, then record it; otherwise, attempt to marshal
		// the JSON data
		if err != nil {
			errMsg := fmt.Sprintf("Could not contact cMix node %s on "+
				"round %d (%d/%d): %+v",
				currentNodeID, roundID, i+1, topology.Len(), err)
			errs = append(errs, errMsg)
		} else {
			err = json.Unmarshal([]byte(metric.RoundMetricJSON), &roundMetric)
			if err != nil {
				errMsg := fmt.Sprintf("Unable to unmarshal response on "+
					"node %s on round %d (%d/%d): %v",
					currentNodeID, roundID, i+1, topology.Len(), err)
				errs = append(errs, errMsg)
			} else {
				roundMetrics[i] = roundMetric
			}
		}
	}

	// If errors occurred above, then concatenate them into a new error to be
	// returned
	var errReturn error
	if len(errs) > 0 {
		errReturn = errors.New(strings.Join(errs, errorDelimiter))
	}

	return roundMetrics, errReturn
}
