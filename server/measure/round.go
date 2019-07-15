package measure

import (
	"gitlab.com/elixxir/server/server/phase"
	"time"
)

type RoundMetrics struct {
	NodeId       string
	Index        uint32
	NumNodes     uint32
	RoundId      uint32
	StartTime    time.Time
	EndTime      time.Time
	PhaseMetrics []PhaseMetrics // Map of phase to metrics
}

type PhaseMetrics struct {
	Phase   string
	Metrics Metrics
}

func NewRoundMetrics(nid string, rid, numNodes uint32) RoundMetrics {
	return RoundMetrics{
		NodeId:   nid,
		RoundId:  rid,
		NumNodes: numNodes,
	}
}

func NewPhaseMetrics(phase string, metrics Metrics) PhaseMetrics {
	return PhaseMetrics{
		Phase:   phase,
		Metrics: metrics,
	}
}

func (rm *RoundMetrics) AddPhase(name phase.Type, metrics Metrics) {
	phaseMetrics := NewPhaseMetrics(name.String(), metrics)
	test := append(rm.PhaseMetrics, phaseMetrics)
}
