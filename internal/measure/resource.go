////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package measure

// measure/resource.go contains the resourceMetric object, the resourceMonitor object.
// These keep track of computational resources while server is running

import (
	"sync"
	"time"
)

// ResourceMetric structure stores memory and thread usage metrics.
type ResourceMetric struct {
	SystemStartTime time.Time
	Time            time.Time
	MemAllocBytes   uint64
	MemAvailable    uint64
	NumThreads      int
	CPUPercentage   float64
}

// ResourceMonitor structure contains a mutable resource metric.
type ResourceMonitor struct {
	lastMetric ResourceMetric
	sync.RWMutex
}

// Get returns a copy of the last ResourceMetric.
func (rm *ResourceMonitor) Get() ResourceMetric {
	rm.RLock()
	defer rm.RUnlock()

	return rm.lastMetric
}

// Set sets the lastMetric of the ResourceMonitor to a copy of the specified
// ResourceMetric.
func (rm *ResourceMonitor) Set(b ResourceMetric) {
	rm.Lock()
	defer rm.Unlock()

	rm.lastMetric = b
}
