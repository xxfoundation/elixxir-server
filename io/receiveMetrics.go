///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package io

// receiveMetrics.go contains the handler for receiveGetMeasure

import (
	"encoding/json"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/measure"
	"time"
)

// ReceiveGetMeasure finds the round in msg and response with a RoundMetrics message
func ReceiveGetMeasure(instance *internal.Instance, msg *mixmessages.RoundInfo) (*mixmessages.RoundMetrics, error) {
	roundID := id.Round(msg.ID)

	rm := instance.GetRoundManager()

	// Check that the round exists, grab it
	r, err := rm.GetRound(roundID)
	if err != nil {
		return nil, err
	}

	t := time.NewTimer(500 * time.Millisecond)
	c := r.GetMeasurementsReadyChan()
	select {
	case <-c:
	case <-t.C:
		return nil, errors.New("Timer expired, could not " +
			"receive measurement")
	}

	// Get data for metrics object
	nodeId := instance.GetID()
	topology := r.GetTopology()
	index := topology.GetNodeLocation(nodeId)
	numNodes := topology.Len()
	resourceMonitor := instance.GetResourceMonitor()

	resourceMetric := measure.ResourceMetric{}

	//fmt.Printf("Resouce monitor: %v", resourceMonitor)
	if resourceMonitor != nil {
		resourceMetric = resourceMonitor.Get()
	}

	metrics := r.GetMeasurements(nodeId, numNodes, index, resourceMetric)

	s, err := json.Marshal(metrics)

	ret := mixmessages.RoundMetrics{
		RoundMetricJSON: string(s),
	}

	return &ret, nil
}
