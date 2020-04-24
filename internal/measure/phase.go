////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package measure

// measure phase.go contains the phaseMetrics object and interface

// phaseMetric structure stores Metrics with an associated phase name.
type phaseMetric struct {
	PhaseName string
	Metrics   Metrics
}

// PhaseMetrics is a list of phaseMetric objects.
type PhaseMetrics []phaseMetric
