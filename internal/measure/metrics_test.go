////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package measure

import (
	"math/rand"
	"reflect"
	"testing"
	"time"
)

// Tests that the Measure() records all the tags with correct timestamps. A list
// of random string are added as events and then the Metrics event list is
// checked to have all the correct tags and timestamps.
func TestMetrics_Measure(t *testing.T) {
	metrics := new(Metrics)

	// Make array of random strings
	testTags := make([]string, rand.Intn(100))
	for i := range testTags {
		testTags[i] = randomString(rand.Intn(100))
	}

	// Record all the random strings as metric events and save their timestamps
	testTimestamps := make([]time.Time, len(testTags))
	for i, value := range testTags {
		testTimestamps[i] = metrics.Measure(value)
	}

	// Check the length of the Metrics Events
	if len(metrics.Events) != len(testTags) {
		t.Errorf("Measure() did not properly record the correct number of "+
			"Metric events\n\texpected: %d\n\treceived: %d",
			len(testTags), len(metrics.Events))
	}

	// Check that all Metric events have the expected tags and timestamps
	for i, metric := range metrics.Events {
		if metric.Tag != testTags[i] {
			t.Errorf("Measure() did not properly record the Metric "+
				"tag on index %d\n\texpected: %s\n\treceived: %s",
				i, testTags[i], metric.Tag)
		}

		if !metric.Timestamp.Equal(testTimestamps[i]) {
			t.Errorf("Measure() did not properly record the Metric "+
				"timestamp on index %d\n\texpected: %s\n\treceived: %s",
				i, metric.Timestamp.String(), testTimestamps[i].String())
		}
	}

	// Check that newer Metric events have a newer timestamp
	for i := 0; i < len(metrics.Events)-1; i++ {
		if metrics.Events[i].Timestamp.After(metrics.Events[i+1].Timestamp) {
			t.Errorf("Measure() did not properly record the Metric "+
				"timestamp. The timestamp of Metric[%d] occured after Metric[%d]"+
				"\n\ttimestamp A: %s\n\ttimestamp B: %s",
				i, i+1, metrics.Events[i].Timestamp.String(),
				metrics.Events[i+1].Timestamp.String())
		}
	}

	// Check that older Metric events have an older timestamp
	for i := 1; i < len(metrics.Events); i++ {
		if metrics.Events[i].Timestamp.Before(metrics.Events[i-1].Timestamp) {
			t.Errorf("Measure() did not properly record the Metric "+
				"timestamp. The timestamp of Metric[%d] occured before Metric[%d]"+
				"\n\ttimestamp A: %s\n\ttimestamp B: %s",
				i, i-1, metrics.Events[i].Timestamp.String(),
				metrics.Events[i-1].Timestamp.String())
		}
	}
}

// Test that Measure() is thread safe by checking if it correctly locks Metrics
// when writing to Events.
func TestMetrics_Measure_Lock(t *testing.T) {
	// Make and lock metric
	metrics := new(Metrics)
	metrics.Lock()

	// Create a new channeled bool to allow another goroutine to write to it to
	// allow a test goroutine to communicate
	result := make(chan bool)

	// Run Measure() with the expectation that it crashes; if it does not, then
	// the result becomes true
	go func() {
		metrics.Measure("test1")
		result <- true
	}()

	// Wait to see if the function does not crash. If it does not, then an error
	// will be printed
	select {
	case <-result:
		t.Error("Measure() did not correctly lock the thread when expected")
	case <-time.After(1 * time.Second):
		return
	}
}

// Tests that the array returned by GetEvents() matches Metrics.Events.
func TestMetrics_GetEvents(t *testing.T) {
	// Create new Metrics and fill Events
	metrics := new(Metrics)
	for i := 0; i < rand.Intn(100); i++ {
		metrics.Measure(randomString(rand.Intn(100)))
	}

	// Get a copy of the events array
	events := metrics.GetEvents()

	// Check to make sure that the returned events match Metrics.Events
	if !reflect.DeepEqual(events, metrics.Events) {
		t.Errorf("GetEvents() did not return a copy of Metrics.Events"+
			"\n\texpected: %v\n\treceived: %v", metrics.Events, events)
	}
}

// Tests that the array returned by GetEvents() is a copy of Metrics.Events.
func TestMetrics_GetEvents_Copy(t *testing.T) {
	// Create new Metrics and add items to Events
	metrics := new(Metrics)
	metrics.Measure("test1")
	metrics.Measure("test2")

	// Get a copy of the events array
	events := metrics.GetEvents()

	// Make change to original array
	metrics.Events[0].Tag = "something else"

	// Check to make sure the change was not reflected in the copy
	if reflect.DeepEqual(events, metrics.Events) {
		t.Errorf("GetEvents() returned the array instead of a copy"+
			"\n\texpected: %v\n\treceived: %v", metrics.Events, events)
	}
}

// Generates a random string.
func randomString(n int) string {
	var letter = []rune(
		"abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	b := make([]rune, n)
	for i := range b {
		b[i] = letter[rand.Intn(len(letter))]
	}
	return string(b)
}
