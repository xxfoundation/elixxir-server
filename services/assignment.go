////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package services

import (
	jww "github.com/spf13/jwalterweatherman"
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
	//location to add new chunks to the waiting list
	waitingIndex *uint32
	//Number of new chunks added to the waiting list
	waitingAdded *uint32
	//Number of assignments completed
	completed *uint32

	//Number of slots per assignment
	numSlots uint32
	//Internal assignment count at which the assignment is ready
	//Generally numSlots*numInputs
	maxCount uint32

	//Threshold at which the module starts operating
	threshold uint32

	// Threshold in chunks at which the module starts operating
	thresholdChunks uint32

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
		jww.FATAL.Panicf("assignment size overflow, Expected: <=%v, "+
			"Got: %v from Weight: %v", maxCount, cnt, weight)
	}

	return cnt == maxCount
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

		completedChunks := uint32(len(cList))

		//Get the updated state of the prime counter
		addingIndex := atomic.AddUint32(al.waitingIndex, completedChunks)

		if addingIndex < al.thresholdChunks {
			for i := addingIndex - completedChunks; i < addingIndex; i++ {
				al.waiting[i] = cList[i-(addingIndex-completedChunks)]
			}

			primed := atomic.AddUint32(al.waitingAdded, completedChunks)

			//If its less then the threshold, store the new chucks and dont return then
			if primed >= al.thresholdChunks {
				cList = append(al.waiting, cList...)
			}
		}
	}

	return cList
}

//Checks if all assignments within a chunk are complete
func (al *assignmentList) DenoteCompleted(numCompleted int) bool {
	result := atomic.AddUint32(al.completed, uint32(numCompleted))
	if result > uint32(len(al.assignments)) {
		jww.FATAL.Panicf("completed more assignments then possible")
	}

	return result == uint32(len(al.assignments))
}
