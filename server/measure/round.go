////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package measure

import (
	"encoding/json"
	"time"
)

// Hold metrics for a round, including phase:metric map
type RoundMetrics struct {
	NodeId       string
	Index        int
	NumNodes     int
	RoundId      uint32
	StartTime    time.Time
	EndTime      time.Time
	PhaseMetrics map[string]Metrics // Map of phase to metrics
}

// Create a RoundMetrics object, taking in node ID, round ID, number of nodes and index
func NewRoundMetrics(nid string, rid uint32, numNodes, nodeIndex int) RoundMetrics {
	return RoundMetrics{
		NodeId:       nid,
		Index:        nodeIndex,
		RoundId:      rid,
		NumNodes:     numNodes,
		PhaseMetrics: map[string]Metrics{},
	}
}

// Add a phase & its metrics to the RoundMetrics object
func (rm *RoundMetrics) AddPhase(name string, metrics Metrics) {
	rm.PhaseMetrics[name] = metrics
}

// Implement Marshaller interface so json.Marshall can be called on RoundMetrics
func (rm *RoundMetrics) MarshallJSON() ([]byte, error) {
	b, err := json.Marshal(rm)
	if err != nil {
		return nil, err
	}
	return b, nil
}
