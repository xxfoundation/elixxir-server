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

// A metric for memory and thread usage
type ResourceMetric struct {
	Time                      time.Time
	MemoryAllocated           string
	MemoryAllocationThreshold int64
	NumThreads                int
	HighestMemThreads         string
}

// Contains a mutable resource metric accessed through a mutex
type ResourceMonitor struct {
	lastMetric *ResourceMetric
	sync.Mutex
}

// Get a resource metric using a lock
func (mh ResourceMonitor) Get() *ResourceMetric {
	mh.Lock()
	defer mh.Unlock()
	return mh.lastMetric
}

// Set a resource metric using a lock
func (mh *ResourceMonitor) Set(rm *ResourceMetric) {
	mh.Lock()
	defer mh.Unlock()
	mh.lastMetric = rm
}
