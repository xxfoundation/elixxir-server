///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package measure

// measure/round.go contains the roundMetrics object, constructors and its methods

import (
	"git.xx.network/xx_network/primitives/id"
	"time"
)

// RoundMetrics structure holds metrics for the life-cycle of a round. It
// includes the events within a phase and resource metrics.
type RoundMetrics struct {
	NodeID         id.ID
	NumNodes       int
	Index          int
	IP             string
	RoundID        id.Round
	BatchSize      uint32
	PhaseMetrics   PhaseMetrics
	ResourceMetric ResourceMetric // Memory and thread usage metrics

	// Special recorded events
	StartTime time.Time
	EndTime   time.Time

	// Round trip data
	RTDurationMilli float64
	RTPayload       string

	// Total dispatch Duration
	DispatchDuration time.Duration
}

// NewRoundMetrics initializes a new RoundMetrics object with the specified
// round ID.
func NewRoundMetrics(roundId id.Round, batchSize uint32) RoundMetrics {
	return RoundMetrics{
		RoundID:      roundId,
		BatchSize:    batchSize,
		StartTime:    time.Now().Round(0),
		PhaseMetrics: PhaseMetrics{},
	}
}

// AddPhase adds a phase and its metrics to the RoundMetrics object.
func (rm *RoundMetrics) AddPhase(name string, metrics Metrics) {
	metrics.NodeId = &rm.NodeID
	newPhaseMetric := phaseMetric{name, metrics}

	rm.PhaseMetrics = append(rm.PhaseMetrics, newPhaseMetric)
}

// SetNodeID sets the node ID for the round metrics.
func (rm *RoundMetrics) SetNodeID(nodeID *id.ID) {
	rm.NodeID = *nodeID.DeepCopy()
}

// SetNumNodes sets the number of nodes for the round metrics.
func (rm *RoundMetrics) SetNumNodes(numNodes int) {
	rm.NumNodes = numNodes
}

// SetIndex sets the node index for the round metrics.
func (rm *RoundMetrics) SetIndex(index int) {
	rm.Index = index
}

// SetResourceMetrics sets the ResourceMetric for the round metrics
func (rm *RoundMetrics) SetResourceMetrics(resourceMetric ResourceMetric) {
	rm.ResourceMetric = resourceMetric
}
