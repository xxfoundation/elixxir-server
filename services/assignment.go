package services

import (
	"fmt"
	"sync/atomic"
)

//Defines a unit of work for a module
type assignment struct {
	//Beginning of the region defined by the chunk
	start uint32

	//atomic compatible count of completion of chunk
	count *uint32
}

//Defines a set of assignments to be processed by a module
type assignmentList struct {
	//List of assignments
	assignments []*assignment
	//Number of assignments ready to be completed
	primed *uint32
	//Number of assignments completed
	completed *uint32

	//Number of slots per assignment
	numSlots uint32
	//Internal assignment count at which the assignment is ready
	//Generally numSlots*numInputs
	maxCount uint32

	//Threshold at which the module starts operating
	threshold uint32

	//WaitingAssignments
	waiting []Chunk
}

//creats and assignment
func newAssignment(start uint32) *assignment {
	var count uint32

	return &assignment{
		start: start,
		count: &count,
	}
}

//Denotes that a portion of an assignment is complete
func (a *assignment) Enqueue(weight, maxCount uint32) bool {

	cnt := atomic.AddUint32(a.count, weight)

	if cnt > maxCount {
		panic(fmt.Sprintf("assignment size overflow, Expected: <=%v, Got: %v from Weight: %v", maxCount, cnt, weight))
	} else {
		return cnt == maxCount
	}
}

// Gets the chunk represented by the assignment
func (a *assignment) GetChunk(size uint32) Chunk {
	return Chunk{a.start, a.start + size}
}

// Denotes all assignments which have completed based upon an incoming chunk and
func (al *assignmentList) PrimeOutputs(c Chunk) []Chunk {
	position := c.Begin()

	var cList []Chunk

	undenotedComplete := c.Len()

	for undenotedComplete > 0 {

		assignmentNum := position / al.numSlots
		weight := (assignmentNum+1)*al.numSlots - position

		if weight > undenotedComplete {
			weight = undenotedComplete
		}

		ready := al.assignments[assignmentNum].Enqueue(weight, al.maxCount)

		if ready {
			cList = append(cList, al.assignments[assignmentNum].GetChunk(al.numSlots))
		}
		position += weight
		undenotedComplete -= weight
	}

	if len(cList) > 0 {
		primed := uint32(0)
		loaded := uint32(0)
		success := false

		cListLen := uint32(len(cList))

		//Get the updated state of the prime counter
		for !success {
			loaded = atomic.LoadUint32(al.primed)
			primed = loaded + cListLen
			success = atomic.CompareAndSwapUint32(al.primed, loaded, primed)
		}

		//If its less then the threshold, store the new chucks and dont return then
		if primed < al.threshold {
			al.waiting = append(al.waiting, cList...)
			cList = make([]Chunk, 0)
			//If this operation crossed the threashold line, return all waiting chunks
		} else if loaded <= al.threshold && primed >= al.threshold {
			cList = append(al.waiting, cList...)
			al.waiting = make([]Chunk, 0)
		}
	}

	return cList
}

//Checks if all assignments within a chunk are complete
func (al *assignmentList) DenoteCompleted(numCompleted int) bool {
	result := atomic.AddUint32(al.completed, uint32(numCompleted))
	if result > uint32(len(al.assignments)) {
		panic("completed more assignments then possible")
	} else {
		return result == uint32(len(al.assignments))
	}
}
