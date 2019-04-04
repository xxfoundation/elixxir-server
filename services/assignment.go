package services

import (
	"fmt"
	"math"
	"sync/atomic"
)

//Defines a unit of work for a module
type assignment struct {
	//Beginning of the region defined by the chunk
	start uint32
	//Size of the assignment
	assignmentSize uint32
	//Size of the chunk
	chunkSize uint32

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

func newAssignment(start, max, assignmentSize, chunkSize uint32) *assignment {
	var count uint32

	return &assignment{
		start:          start,
		count:          &count,
		maxCount:       max,
		assignmentSize: assignmentSize,
		chunkSize:      chunkSize,
	}
}

func (a *assignment) Enqueue(weight uint32) bool {
	end := false
	var ready bool

	for !end {
		cntOld := atomic.LoadUint32(a.count)
		cnt := cntOld + weight

		if cnt > a.maxCount {
			panic(fmt.Sprintf("assignment assignmentSize overflow, Expected: <=%v, Got: %v from Weight: %v off %v", a.maxCount, cnt, weight, cntOld))
		} else if cnt == a.maxCount {
			ready = true
		} else {
			ready = false
		}

		end = atomic.CompareAndSwapUint32(a.count, cntOld, cnt)
	}

	return ready
}

func (a *assignment) GetChunk() []Chunk {
	var chunks []Chunk
	for p := a.start; p < a.start+uint32(a.assignmentSize); p += a.chunkSize {
		chunks = append(chunks, Chunk{p, p + a.chunkSize})
	}
	return chunks
}

// This method name doesn't seem quite right
func (al *assignmentList) PrimeOutputs(c Chunk) ([]Chunk, int) {
	position := c.Begin()

	var cList []Chunk

	denoted := c.Len()

	numComplete := 0

	for denoted > 0 {
		assignmentNum := uint32(math.Floor(float64(position) / float64(al.assignmentSize)))
		weight := (assignmentNum+1)*al.assignmentSize - position

		if weight > denoted {
			weight = denoted
		}

		ready := al.assignments[assignmentNum].Enqueue(weight)

		if ready {
			numComplete++
			cList = append(cList, al.assignments[assignmentNum].GetChunk()...)
		}
		position += weight
		denoted -= weight
	}
	return cList, numComplete
}

func (al *assignmentList) DenoteCompleted(numCompleted int) bool {
	swapComplete := false

	done := false

	nc := uint32(numCompleted)

	for !swapComplete {
		completedOld := atomic.LoadUint32(al.assignmentsCompleted)
		completed := completedOld + nc

		if completed == uint32(len(al.assignments)) {
			done = true
		} else if completed > uint32(len(al.assignments)) {
			panic("completed more assignments then possible")
		}

		swapComplete = atomic.CompareAndSwapUint32(al.assignmentsCompleted, completedOld, completed)
	}

	return done
}
