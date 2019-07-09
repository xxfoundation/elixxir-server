package measure

import (
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
		t.Errorf("Metric recorded invalid timestamp for event 0.\r\nExpected timestamp to be between the before and after timestamps.\r\nBefore: %s\r\nGot: %s\r\nAfter: %s",
			before.String(), metrics.Events[0].Timestamp.String(), after.String())
	}

	if metrics.Events[1].Timestamp.After(before) != true && metrics.Events[1].Timestamp.Before(after) != true {
		t.Errorf("Metric recorded invalid timestamp for event 1.\r\nExpected timestamp to be between the before and after timestamps.\r\nBefore: %s\r\nGot: %s\r\nAfter: %s",
			before.String(), metrics.Events[1].Timestamp.String(), after.String())
	}
}

// Test mutex lock properly locks to make sure function is thread safe
func TestMeasureLock(t *testing.T) {
	// Make and lock metric
	metrics := new(Metrics)
	metrics.Lock()

	// Create a new channeled bool to allow another goroutine to write to it.
	// This way a test goroutine can communicate back to us.
	result := make(chan bool)

	// Run measure in a new goroutine and hope it crashes, if it doesn't the
	// result becomes true. This is bad because we want the function to crash,
	// we previously write locked the metrics struct (and therefore it's Events
	// array) so other goroutines can't write to it.
	go func() {
		metrics.Measure("test1")
		result <- true
	}()

	// We wait a second to see if the function does write true to the result var.
	// If it does, it did not panic (because the mutex lock didn't work), which is bad.
	select {
	case <-result:
		t.Error("Measure() does not correctly lock thread")
	case <-time.After(1 * time.Second):
		return
	}
}
