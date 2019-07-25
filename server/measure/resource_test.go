////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package measure

import (
	"testing"
	"time"
)

// Tests that the getter and setter of a
// resource monitor works
func TestResourceMetric(t *testing.T) {

	expectedResourceMetric := ResourceMetric{
		Time:                      time.Unix(1, 2),
		MemoryAllocated:           "1",
		MemoryAllocationThreshold: int64(2),
		NumThreads:                3,
		HighestMemThreads:         "someFuncName",
	}
	resourceMonitor := ResourceMonitor{}

	resourceMonitor.Set(&expectedResourceMetric)

	actualResourceMetric := resourceMonitor.Get()

	if !resourceMetricEq(expectedResourceMetric, *actualResourceMetric) {
		t.Errorf("Resource metric did not match expected value %v", expectedResourceMetric)
	}
}

// Compares the equality of two resource metrics and returns
// true if they are equal and false otherwise
func resourceMetricEq(a ResourceMetric, b ResourceMetric) bool {
	if !a.Time.Equal(b.Time) {
		return false
	} else if a.MemoryAllocated != b.MemoryAllocated {
		return false
	} else if a.MemoryAllocationThreshold != b.MemoryAllocationThreshold {
		return false
	} else if a.NumThreads != b.NumThreads {
		return false
	} else if a.HighestMemThreads != b.HighestMemThreads {
		return false
	}
	return true
}
