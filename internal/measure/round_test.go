////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package measure

import (
	"gitlab.com/elixxir/primitives/id"
	"reflect"
	"testing"
	"time"
)

// Tests that NewRoundMetrics() correctly initialises RoundID, StartTime, and
// PhaseMetrics.
func TestNewRoundMetrics(t *testing.T) {
	roundID := id.Round(42)
	rm := NewRoundMetrics(roundID, 25)

	if rm.RoundID != roundID {
		t.Errorf("NewRoundMetrics() incorrectly setRoundId"+
			"\n\texpected: %v\n\treceived: %v",
			roundID, rm.RoundID)
	}

	if rm.StartTime.After(time.Now()) {
		t.Errorf("NewRoundMetrics() incorrectly StartTime"+
			"\n\texpected: %v\n\treceived: %v",
			time.Now(), rm.StartTime)
	}

	if !reflect.DeepEqual(rm.PhaseMetrics, PhaseMetrics{}) {
		t.Errorf("NewRoundMetrics() incorrectly PhaseMetrics"+
			"\n\texpected: %v\n\treceived: %v",
			PhaseMetrics{}, rm.PhaseMetrics)
	}
}

// Tests AddPhase() by adding a bunch of phases and then comparing the
// PhaseMetrics to the expected array.
func TestRoundMetrics_AddPhase(t *testing.T) {
	// Build PhaseMetrics objects and put into array
	metricA := Metric{"matching_tag", time.Unix(1, 2)}
	metricB := Metric{"matching_tag", time.Unix(1, 3)}
	metricC := Metric{"another_tag2", time.Unix(1, 2)}
	metricD := Metric{}

	metricsA := Metrics{Events: []Metric{metricA, metricB, metricC, metricD}, NodeId: id.NewIdFromBytes([]byte{}, t)}
	metricsB := Metrics{Events: []Metric{metricA, metricB, metricD, metricC}, NodeId: id.NewIdFromBytes([]byte{}, t)}
	metricsC := Metrics{Events: []Metric{}, NodeId: id.NewIdFromBytes([]byte{}, t)}

	phaseMetricA := phaseMetric{"PhaseName1", metricsA}
	phaseMetricB := phaseMetric{"PhaseName1", metricsB}
	phaseMetricC := phaseMetric{"PhaseName2", metricsA}
	phaseMetricD := phaseMetric{"", metricsC}

	pmArr := PhaseMetrics{phaseMetricA, phaseMetricB, phaseMetricC, phaseMetricD}

	// Create new RoundMetrics
	rm := NewRoundMetrics(42, 55)

	// Add all phases to the RoundMetrics PhaseMetrics
	for _, pm := range pmArr {
		rm.AddPhase(pm.PhaseName, pm.Metrics)
	}

	// Check if the PhaseMetrics matches the expected array
	if !reflect.DeepEqual(rm.PhaseMetrics, pmArr) {
		t.Errorf("AddPhase() incorrectly appended to PhaseMetrics"+
			"\n\texpected: %v\n\treceived: %v",
			pmArr, rm.PhaseMetrics)
	}
}

// Tests SetNodeID() by setting the Node ID and checking its value.
func TestRoundMetrics_SetNodeID(t *testing.T) {
	// Create new RoundMetrics
	rm := NewRoundMetrics(42, 34)

	// Set the Node ID
	nodeID := id.NewIdFromString("test", id.Node, t)
	rm.SetNodeID(nodeID)

	// Check if the Node ID was set correctly
	if !rm.NodeID.Cmp(nodeID) {
		t.Errorf("SetNodeID() incorrectly set NodeID"+
			"\n\texpected: %v\n\treceived: %v",
			nodeID, rm.NodeID)
	}
}

// Tests SetNumNodes() by setting NumNodes and checking its value.
func TestRoundMetrics_SetNumNodes(t *testing.T) {
	// Create new RoundMetrics
	rm := NewRoundMetrics(42, 12)

	// Set the Node ID
	numNodes := 42
	rm.SetNumNodes(numNodes)

	// Check if the Node ID was set correctly
	if rm.NumNodes != numNodes {
		t.Errorf("SetNumNodes() incorrectly set NumNodes"+
			"\n\texpected: %v\n\treceived: %v",
			numNodes, rm.NumNodes)
	}
}

// Tests SetIndex() by setting Index and checking its value.
func TestRoundMetrics_SetIndex(t *testing.T) {
	// Create new RoundMetrics
	rm := NewRoundMetrics(42, 89)

	// Set the Node ID
	index := 42
	rm.SetIndex(index)

	// Check if the Node ID was set correctly
	if rm.Index != index {
		t.Errorf("SetIndex() incorrectly set Index"+
			"\n\texpected: %v\n\treceived: %v",
			index, rm.Index)
	}
}

// Tests SetResourceMetrics() by setting Index and checking its value.
func TestRoundMetrics_SetResourceMetrics(t *testing.T) {
	// Create new RoundMetrics
	rm := NewRoundMetrics(42, 34)

	// Set the Node ID
	resourceMetric := ResourceMetric{
		Time:          time.Unix(1, 2),
		MemAllocBytes: 1,
		NumThreads:    3,
	}
	rm.SetResourceMetrics(resourceMetric)

	// Check if the Node ID was set correctly
	if !reflect.DeepEqual(rm.ResourceMetric, resourceMetric) {
		t.Errorf("SetResourceMetrics() incorrectly set ResourceMetric"+
			"\n\texpected: %v\n\treceived: %v",
			resourceMetric, rm.ResourceMetric)
	}
}
