////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package measure

import "time"

type MemMetric struct {
	Time                      time.Time
	MemoryAllocated           string
	MemoryAllocationThreshold int64
	NumThreads                int
	HighestMemThreads         string
}
