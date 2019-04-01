package dispatch

import (
	"fmt"
	"math"
	"sync/atomic"
)

//Defines a unit of work for a module
type assignment struct {
	//Beginning of the region defined by the chunk
	start uint32
	//Size of the chunk
	size uint32

	//atomic compatible count of completion of chunk
	count *uint32
	//value of count at which the chunk is complete
	maxCount uint32
}

//Defines a set of assignments to be processed by a module
type assignmentList struct {
	assignments          []*assignment
	assignmentSize       uint32
	assignmentsCompleted *uint32
}

func newAssignment(start uint32, max uint32, size uint32) *assignment {
	var count uint32

	return &assignment{
		start:    start,
		count:    &count,
		maxCount: max,
		size:     size,
	}
}

func (a *assignment) Enqueue(weight uint32) bool {
	end := false
	var ready bool

	for !end {
		cntOld := atomic.LoadUint32(a.count)
		cnt := cntOld + weight

		if cnt > a.maxCount {
			panic(fmt.Sprintf("assignment size overflow, Expected: <=%v, Got: %v from Weight: %v off %v", a.maxCount, cnt, weight, cntOld))
		} else if cnt == a.maxCount {
			ready = true
		} else {
			ready = false
		}

		end = atomic.CompareAndSwapUint32(a.count, cntOld, cnt)
	}

	return ready
}

func (a *assignment) GetChunk() Lot {
	return Lot{a.start, a.start + uint32(a.size)}
}

// This method name doesn't seem quite right
func (al *assignmentList) PrimeOutputs(c Lot) ([]Lot, bool) {
	position := c.Begin()

	var cList []Lot

	done := false

	denoted := c.Len()

	for denoted > 0 {
		assignmentNum := uint32(math.Floor(float64(position) / float64(al.assignmentSize)))
		weight := (assignmentNum+1)*al.assignmentSize - position

		if weight > denoted {
			weight = denoted
		}

		ready := al.assignments[assignmentNum].Enqueue(weight)

		if ready {

			cList = append(cList, al.assignments[assignmentNum].GetChunk())

			swapComplete := false

			for !swapComplete {
				completedOld := atomic.LoadUint32(al.assignmentsCompleted)
				completed := completedOld + 1

				if completed == uint32(len(al.assignments)) {
					done = true
				}

				swapComplete = atomic.CompareAndSwapUint32(al.assignmentsCompleted, completedOld, completed)
			}
		}
		position += weight
		denoted -= weight
	}
	return cList, done
}
