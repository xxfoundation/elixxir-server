package measure

import (
	"sync"
	"time"
)

// Metrics struct holds list of different metrics for the phase. RWMutex allows
// to be read in multiple threads at once, but only allows one writen in
// one at a time
type Metrics struct{
	Events []Metric
	sync.RWMutex
}

// Metric struct holds a measurement, containing a phase tag and a timestamp
type Metric struct {
	Tag       string
	Timestamp time.Time
}

// Create a new Metric and add it to the Metrics variable of the phase
func (m *Metrics) Measure(tag string) time.Time {
	// Create new Metric instance
	measure := Metric{
		Tag:       tag,
		Timestamp: time.Now(),
	}

	// Lock Events so other goroutines can't write and mangle data, then append
	m.Lock()
	m.Events = append(m.Events, measure)
	m.Unlock()

	return measure.Timestamp
}