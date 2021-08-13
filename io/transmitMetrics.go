///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Package io transmitMetrics.go handles the endpoints and helper functions for
// receiving and sending the metrics message between cMix nodes.

package io

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	pb "git.xx.network/elixxir/comms/mixmessages"
	"git.xx.network/elixxir/comms/node"
	"git.xx.network/elixxir/server/internal/measure"
	"git.xx.network/xx_network/comms/connect"
	"git.xx.network/xx_network/primitives/id"
	"strings"
)

const errorDelimiter = "; "

// Get metrics for all nodes in the topology, returning a JSON map of server
// address to metrics.
func TransmitGetMeasure(network *node.Comms, topology *connect.Circuit, roundID id.Round) ([]measure.RoundMetrics, error) {

	// Stores errors for each SendGetMeasure() call to be concatenated on return
	var errs []string

	// Contact all visible servers and get metrics
	roundMetrics := make([]measure.RoundMetrics, topology.Len())

	// Loop through all the nodes
	for i := 0; i < topology.Len(); i++ {
		// Pull the particular server host object from the commManager
		currentNodeID := topology.GetNodeAtIndex(i).String()
		currentNode := topology.GetHostAtIndex(i)
		roundMetric := measure.RoundMetrics{}

		metric, err := network.SendGetMeasure(currentNode, &pb.RoundInfo{
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
