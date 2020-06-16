///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package measure

// measure phase.go contains the phaseMetrics object and interface

// phaseMetric structure stores Metrics with an associated phase name.
type phaseMetric struct {
	PhaseName string
	Metrics   Metrics
}

// PhaseMetrics is a list of phaseMetric objects.
type PhaseMetrics []phaseMetric
