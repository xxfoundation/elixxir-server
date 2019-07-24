////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"encoding/json"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/id"
	"testing"
)

// Mock an implementation with a GetMeasure function
func MockGetMeasureImplementation() *node.Implementation {
	impl := node.NewImplementation()
	impl.Functions.GetMeasure = func(message *mixmessages.RoundInfo) (*mixmessages.RoundMetrics, error) {
		mock := mixmessages.RoundMetrics{
			RoundMetricJSON: "this is totally a json",
		}
		return &mock, nil
	}
	return impl
}

// Test the TransmitGetMeasure function to ensure that all nodes return their metrics
func TestTransmitGetMeasure(t *testing.T) {
	// Mock implementation with a dummy GetMeasure function
	// (this does not actually get any metrics, but that's tested elsewhere)
	impl := MockGetMeasureImplementation()

	// Setup the network
	comms, topology := buildTestNetworkComponents(
		[]*node.Implementation{impl, impl, impl}, 10)
	defer Shutdown(comms)

	// Run the function (round ID doesn't matter bc we mocked GetMeasure)
	rndID := id.Round(2)
	s := TransmitGetMeasure(comms[0], topology, rndID)

	serverMetrics := map[string]string{}
	err := json.Unmarshal([]byte(s), &serverMetrics)

	if err != nil {
		t.Errorf("Error unmarshalling server metrics JSON: %+v", err)
	}

	// Check that all nodes gave back metrics
	if len(serverMetrics) != 3 {
		t.Error("Did not receive metrics from all nodes")
	}
}
