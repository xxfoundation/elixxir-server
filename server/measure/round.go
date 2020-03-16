////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package measure

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/primitives/id"
	"time"
)

// RoundMetrics structure holds metrics for the life-cycle of a round. It
// includes the events within a phase and resource metrics.
type RoundMetrics struct {
	NodeID         string
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
}

// NewRoundMetrics initializes a new RoundMetrics object with the specified
// round ID.
func NewRoundMetrics(roundId id.Round, batchSize uint32, nodeId *id.Node) RoundMetrics {
	return RoundMetrics{
		NodeID:       nodeId.String(),
		RoundID:      roundId,
		BatchSize:    batchSize,
		StartTime:    time.Now().Round(0),
		PhaseMetrics: PhaseMetrics{},
	}
}

// AddPhase adds a phase and its metrics to the RoundMetrics object.
func (rm *RoundMetrics) AddPhase(name string, metrics Metrics) {
	metrics.NodeId = rm.NodeID
	jww.FATAL.Printf("our node id: %+v", rm.NodeID)
	newPhaseMetric := phaseMetric{name, metrics, rm.NodeID}

	rm.PhaseMetrics = append(rm.PhaseMetrics, newPhaseMetric)
}

// SetNodeID sets the node ID for the round metrics.
func (rm *RoundMetrics) SetNodeID(nodeID string) {
	rm.NodeID = nodeID
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
