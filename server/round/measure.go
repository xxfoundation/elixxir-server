package round

import "gitlab.com/elixxir/server/server/measure"

func (r *Round) getMeasurements() measure.RoundMetrics {
	metrics := measure.NewRoundMetrics("temp", uint32(r.id), 54)

	for k, v := range r.phaseMap {
		phaseId := k.String()
		phaseMetrics := r.phases[v]
		phaseMetrics.GetMeasure()
		metrics.AddPhase(phaseId)
	}
}
