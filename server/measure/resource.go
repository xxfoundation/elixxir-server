////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package measure

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
	lastMetric *ResourceMetric
	sync.RWMutex
}

// Get returns the last ResourceMetric.
func (rm ResourceMonitor) Get() *ResourceMetric {
	rm.RLock()
	lastResourceMetric := rm.lastMetric
	rm.RUnlock()

	return lastResourceMetric
}

// Set sets the lastMetric of the ResourceMonitor to the specified
// ResourceMetric.
func (rm *ResourceMonitor) Set(b *ResourceMetric) {
	rm.Lock()
	rm.lastMetric = b
	rm.Unlock()
}
