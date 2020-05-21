////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package measure

// metrics.go contains the metrics object and its methods

import (
	"gitlab.com/elixxir/primitives/id"
	"sync"
	"time"
)

// Metrics structure holds the list of different metrics for the phase. The
// RWMutex prevents two threads from writing to the list at the same time.
type Metrics struct {
	Events []Metric
	NodeId *id.ID
	sync.RWMutex
}

// Metric structure holds a single measurement, which contains a phase tag and a
// timestamp from when the measurement was taken.
type Metric struct {
	Tag       string
	Timestamp time.Time
}

// Measure creates a new Metric object and appends it to the Metrics's event
// list. The Metric object is created from the specified tag and a timestamp
// created at the time of function call. The timestamp is returned.
func (ms *Metrics) Measure(tag string) time.Time {
	// Create new Metric object from the tag and new timestamp
	metric := Metric{
		Tag:       tag,
		Timestamp: time.Now(),
	}

	// Append the metric to the even list
	ms.Lock()
	ms.Events = append(ms.Events, metric)
	ms.Unlock()

	return metric.Timestamp
}

// GetEvents returns a copy of the Events array.
func (ms Metrics) GetEvents() []Metric {
	ms.Lock()
	defer ms.Unlock()
	metricsEvents := make([]Metric, len(ms.Events))

	copy(metricsEvents, ms.Events)

	return metricsEvents
}
