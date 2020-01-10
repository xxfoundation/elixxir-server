////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package services

import (
	"fmt"
	"github.com/pkg/errors"
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
func (a *assignment) Enqueue(weight, maxCount uint32) (bool, error) {

	cnt := atomic.AddUint32(a.count, weight)

	if cnt > maxCount {
		return false, errors.New(fmt.Sprintf("assignment size overflow, Expected: <=%v, "+
			"Got: %v from Weight: %v", maxCount, cnt, weight))
	}

	return cnt == maxCount, nil
}

// Gets the chunk represented by the assignment
func (a *assignment) GetChunk(size uint32) Chunk {
	return Chunk{a.start, a.start + size}
}

// Denotes all assignments which have completed based upon an incoming chunk
func (al *assignmentList) PrimeOutputs(c Chunk) ([]Chunk, error) {
	position := c.Begin()

	var cList []Chunk

	undenotedComplete := c.Len()

	for undenotedComplete > 0 {

		assignmentNum := position / al.numSlots
		weight := (assignmentNum+1)*al.numSlots - position

		if weight > undenotedComplete {
			weight = undenotedComplete
		}

		ready, err := al.assignments[assignmentNum].Enqueue(weight, al.maxCount)

		if err != nil {
			return nil, err
		}

		if ready {
			cList = append(cList, al.assignments[assignmentNum].GetChunk(al.numSlots))
		}
		position += weight
		undenotedComplete -= weight
	}

	//Process the threshold
	//Get the updated state of the prime counter
	addingIndex := atomic.AddUint32(al.waitingIndex, uint32(len(cList)))
	initialChunk := addingIndex - uint32(len(cList))
	if initialChunk*al.numSlots < al.threshold {
		for index, chunk := range cList {
			al.waiting[initialChunk+uint32(index)] = chunk
		}

		totalWaiting := atomic.AddUint32(al.waitingAdded, uint32(len(cList)))

		if totalWaiting*al.numSlots >= al.threshold {
			cList = al.waiting[:totalWaiting]
		} else {
			cList = make([]Chunk, 0)
		}
	}

	return cList, nil
}

//Checks if all assignments within a chunk are complete
func (al *assignmentList) DenoteCompleted(numCompleted int) (bool, error) {

	result := atomic.AddUint32(al.completed, uint32(numCompleted))
	//fmt.Println("denoting complete:", result, "/", len(al.assignments))
	if result > uint32(len(al.assignments)) {
		return false, errors.New(fmt.Sprintf("completed more assignments then possible:"+
			" %d > %d", result, len(al.assignments)))
	}

	return result == uint32(len(al.assignments)), nil
}
