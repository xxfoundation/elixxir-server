package measure

import (
	"fmt"
	"testing"
	"time"
)

// Tests that the measure function writes sane values for each metric
func TestMeasure(t *testing.T) {
	metrics := new(Metrics)

	before := time.Now()
	metrics.Measure("test1")
	metrics.Measure("test2")
	after := time.Now()

	if len(metrics.Events) != 2 {
		t.Error("2 metrics were not recorded")
	}

	if (metrics.Events[0].Tag != "test1") && (metrics.Events[1].Tag != "test2") {
		t.Error("Metric tags were not recorded correctly")
	}

	if metrics.Events[0].Timestamp.After(before) != true && metrics.Events[0].Timestamp.Before(after) != true {
		fmt.Printf("%s, %s, %s\n", before.String(), metrics.Events[0].Timestamp.String(), after.String())
		t.Error("Metric recorded invalid timestamp for event 0")
	}

	if metrics.Events[1].Timestamp.After(before) != true && metrics.Events[1].Timestamp.Before(after) != true {
		fmt.Printf("%s, %s, %s\n", before.String(), metrics.Events[1].Timestamp.String(), after.String())
		t.Error("Metric recorded invalid timestamp for event 1")
	}
}

// Test mutex lock properly locks to make sure function is thread safe
func TestMeasureLock(t *testing.T) {
	metrics := new(Metrics)
	metrics.Lock()

	result := make(chan bool)

	go func() {
		metrics.Measure("test1")
		result <- true
	}()

	select {
	case <-result:
		t.Error("Measure() does not correctly lock thread")
	case <-time.After(1 * time.Second):
		return
	}
}
