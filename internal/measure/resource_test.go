///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package measure

import (
	"math/rand"
	"reflect"
	"testing"
	"time"
)

// Tests that Get() retrieves the correct ResourceMetric from a ResourceMonitor.
func TestResourceMonitor_Get(t *testing.T) {
	expectedResourceMetric := ResourceMetric{
		SystemStartTime: time.Now(),
		Time:            time.Now().Add(time.Minute),
		MemAllocBytes:   uint64(rand.Int63n(100)),
		MemAvailable:    uint64(rand.Int63n(100)),
		NumThreads:      rand.Intn(100),
		CPUPercentage:   rand.Float64() * 100,
	}

	resourceMonitor := ResourceMonitor{lastMetric: expectedResourceMetric}

	if !reflect.DeepEqual(expectedResourceMetric, resourceMonitor.Get()) {
		t.Errorf("Get() returned an incorrect ResourceMetric"+
			"\n\texpected: %v\n\treceived: %v",
			expectedResourceMetric, resourceMonitor.Get())
	}
}

// TODO: write test that tests if the pointer was copied instead of data

// Tests that Set() sets the correct ResourceMetric to a ResourceMonitor.
func TestResourceMonitor_Set(t *testing.T) {
	expectedResourceMetric := ResourceMetric{
		SystemStartTime: time.Now(),
		Time:            time.Now().Add(time.Minute),
		MemAllocBytes:   uint64(rand.Int63n(100)),
		MemAvailable:    uint64(rand.Int63n(100)),
		NumThreads:      rand.Intn(100),
		CPUPercentage:   rand.Float64() * 100,
	}

	resourceMonitor := ResourceMonitor{}
	resourceMonitor.Set(expectedResourceMetric)

	if !reflect.DeepEqual(expectedResourceMetric, resourceMonitor.lastMetric) {
		t.Errorf("Set() set an incorrect ResourceMetric"+
			"\n\texpected: %v\n\treceived: %v",
			expectedResourceMetric, resourceMonitor.lastMetric)
	}
}

// Test that Set() is thread safe by checking if it correctly locks the
// ResourceMonitor when writing to lastMetric.
func TestResourceMonitor_Set_Lock(t *testing.T) {
	// Make and lock the ResourceMonitor
	expectedResourceMetric := ResourceMetric{
		SystemStartTime: time.Now(),
		Time:            time.Now().Add(time.Minute),
		MemAllocBytes:   uint64(rand.Int63n(100)),
		MemAvailable:    uint64(rand.Int63n(100)),
		NumThreads:      rand.Intn(100),
		CPUPercentage:   rand.Float64() * 100,
	}

	resourceMonitor := ResourceMonitor{}
	resourceMonitor.Lock()

	// Create a new channeled bool to allow another goroutine to write to it to
	// allow a test goroutine to communicate
	result := make(chan bool)

	// Run Set() with the expectation that it crashes; if it does not, then
	// the result becomes true
	go func() {
		resourceMonitor.Set(expectedResourceMetric)
		result <- true
	}()

	// Wait to see if the function does not crash. If it does not, then an error
	// will be printed
	select {
	case <-result:
		t.Error("Set() did not correctly lock the thread when expected")
	case <-time.After(1 * time.Second):
		return
	}
}

// Test that a copy is created.
func TestResourceMetric_Copy(t *testing.T) {
	expectedResourceMetric := ResourceMetric{
		SystemStartTime: time.Now(),
		Time:            time.Now().Add(time.Minute),
		MemAllocBytes:   uint64(rand.Int63n(100)),
		MemAvailable:    uint64(rand.Int63n(100)),
		NumThreads:      rand.Intn(100),
		CPUPercentage:   rand.Float64() * 100,
	}

	copyResourceMetric := expectedResourceMetric

	if !reflect.DeepEqual(expectedResourceMetric, copyResourceMetric) {
		t.Errorf("copy() did not copy the correct values"+
			"\n\texpected: %v\n\treceived: %v",
			expectedResourceMetric, copyResourceMetric)
	}

	if &expectedResourceMetric == &copyResourceMetric {
		t.Errorf("copy() did not make a copy; the pointers are the same"+
			"\n\texpected: %p\n\treceived: %p",
			&expectedResourceMetric, &copyResourceMetric)
	}
}
