////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"encoding/json"
	"errors"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/id"
	"testing"
)

const mockRoundMetricJSON = `{
	"NodeID": "abc",
	"NumNodes": 3,
	"Index": 0,
	"IP": "0.0.0.0",
	"RoundID": 4,
	"BatchSize": 0,
	"PhaseMetrics": [],
	"ResourceMetric": {
		"SystemStartTime": "0001-01-01T00:00:00Z",
		"Time": "0001-05-01T00:00:00Z",
		"MemAllocBytes": 5,
		"MemAvailable": 13,
		"NumThreads": 5,
		"CPUPercentage": 0
	},
	"StartTime": "0001-01-01T00:00:00Z",
	"EndTime": "0001-02-03T00:00:00Z",
	"RTDurationMilli": 0,
	"RTPayload": ""
}`

// Mock an implementation with a GetMeasure function.
func MockGetMeasureImplementation(mockJSON string, err error) *node.Implementation {
	impl := node.NewImplementation()

	impl.Functions.GetMeasure = func(message *mixmessages.RoundInfo, auth *connect.Auth) (*mixmessages.RoundMetrics, error) {
		mock := mixmessages.RoundMetrics{
			RoundMetricJSON: mockJSON,
		}

		return &mock, err
	}

	return impl
}

// Test the TransmitGetMeasure() function to ensure that all nodes return their
// metrics.
func TestTransmitGetMeasure(t *testing.T) {
	// Mock implementation with a dummy GetMeasure function (this does not
	// actually get any metrics, but that's tested elsewhere)
	impl := MockGetMeasureImplementation(mockRoundMetricJSON, nil)

	// Setup the network
	comms, topology := buildTestNetworkComponents(
		[]*node.Implementation{impl, impl, impl}, 10)
	defer Shutdown(comms)

	// Run the function (round ID does not matter because we mocked GetMeasure)
	s, err := TransmitGetMeasure(comms[0], topology, id.Round(2))

	if err != nil {
		t.Errorf("TransmitGetMeasure() unexpectedly returned an error"+
			"\n\terror: %+v", err)
	}

	data, err := json.MarshalIndent(s[0], "", "\t")
	if err != nil {
		t.Errorf("Unexpected error when unmarshalling server metrics JSON"+
			"\n\terror: %+v", err)
	}

	if string(data) != mockRoundMetricJSON {
		t.Errorf("TransmitGetMeasure() did not return the correct "+
			"RoundMetrics\n\texpected: %s\n\treceived: %s",
			mockRoundMetricJSON, string(data))
	}
}

// Test that TransmitGetMeasure() errors correctly.
func TestTransmitGetMeasure_Error(t *testing.T) {
	// Mock implementation with a dummy GetMeasure function (this does not
	// actually get any metrics, but that's tested elsewhere)
	impl := MockGetMeasureImplementation(mockRoundMetricJSON,
		errors.New("TEST"))

	// Setup the network
	comms, topology := buildTestNetworkComponents(
		[]*node.Implementation{impl, impl, impl}, 10)
	defer Shutdown(comms)

	// Run the function (round ID does not matter because we mocked GetMeasure)
	rndID := id.Round(2)
	_, err := TransmitGetMeasure(comms[0], topology, rndID)

	if err == nil {
		t.Errorf("TransmitGetMeasure() unexpectedly did not return an error")
	}
}

// Test that TransmitGetMeasure() errors correctly with malformed JSON.
func TestTransmitGetMeasure_JSONError(t *testing.T) {
	// Mock implementation with a dummy GetMeasure function (this does not
	// actually get any metrics, but that's tested elsewhere)
	impl := MockGetMeasureImplementation(mockRoundMetricJSON+"aw9awd", nil)

	// Setup the network
	comms, topology := buildTestNetworkComponents(
		[]*node.Implementation{impl, impl, impl}, 10)
	defer Shutdown(comms)

	// Run the function (round ID does not matter because we mocked GetMeasure)
	rndID := id.Round(2)
	_, err := TransmitGetMeasure(comms[0], topology, rndID)

	if err == nil {
		t.Errorf("TransmitGetMeasure() unexpectedly did not return an error")
	}
}
