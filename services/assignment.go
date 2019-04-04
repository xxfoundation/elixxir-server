package services

import (
	"fmt"
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

	cnt := atomic.AddUint32(a.count, weight)

	if cnt > a.maxCount {
		panic(fmt.Sprintf("assignment assignmentSize overflow, Expected: <=%v, Got: %v from Weight: %v", a.maxCount, cnt, weight))
	} else {
		return cnt == a.maxCount
	}
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

	undenotedComplete := c.Len()

	numComplete := 0

	for undenotedComplete > 0 {
		assignmentNum := position / al.assignmentSize;
		weight := (assignmentNum+1)*al.assignmentSize - position

		if weight > undenotedComplete {
			weight = undenotedComplete
		}

		ready := al.assignments[assignmentNum].Enqueue(weight)

		if ready {
			numComplete++
			cList = append(cList, al.assignments[assignmentNum].GetChunk()...)
		}
		position += weight
		undenotedComplete -= weight
	}
	return cList, numComplete
}

func (al *assignmentList) DenoteCompleted(numCompleted int) bool {

	result := atomic.AddUint32(al.assignmentsCompleted, uint32(numCompleted))
	if result > uint32(len(al.assignments)) {
		panic("completed more assignments then possible")
	} else {
		return result == uint32(len(al.assignments))
	}
}
