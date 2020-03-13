////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package measure

// phaseMetric structure stores Metrics with an associated phase name.
type phaseMetric struct {
	PhaseName string
	Metrics   Metrics
	nodeId    string
}

// PhaseMetrics is a list of phaseMetric objects.
type PhaseMetrics []phaseMetric
