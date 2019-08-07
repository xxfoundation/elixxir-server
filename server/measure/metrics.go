package measure

import (
	jww "github.com/spf13/jwalterweatherman"
	"os"
	"sync"
	"time"
)

// Metrics struct holds list of different metrics for the phase. RWMutex allows
// to be read in multiple threads at once, but only allows one writen in
// one at a time
type Metrics struct {
	Events []Metric
	sync.RWMutex
}

// Metric struct holds a measurement, containing a phase tag and a timestamp
type Metric struct {
	Tag       string
	Timestamp time.Time
}

// GetEvents returns a copy of the Events array, containing all Metric events
func (m Metrics) GetEvents() []Metric {
	metricsArray := make([]Metric, len(m.Events))

	for i := 0; i <= len(m.Events)-1; i++ {
		metricsArray[i] = Metric(m.Events[i])
	}

	return metricsArray
}

// Measure creates a new Metric and add it to the Metrics variable of the phase
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

// AppendToMetricsLog appends a measures to
// a log file which is located in logPath
func AppendToMetricsLog(logPath, measures string) {
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		jww.WARN.Printf("Unable to open metrics log file")
		return
	}
	defer func() {
		if f != nil {
			_ = f.Close()
		}
	}()

	_, err = f.WriteString(measures)
	if err != nil {
		jww.WARN.Printf("Unable to append to metrics log file")
	}
}