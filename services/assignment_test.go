package services

import "testing"


func TestAssignment_Enqueue(t *testing.T) {
	a := newAssignment(100)
	k, err := a.Enqueue(102,101)

	if err != nil{
		//error here should equal nil
	}
}

func TestAssignment_GetChunk(t *testing.T) {

}

func TestAssignmentList_DenoteCompleted(t *testing.T) {

}

func TestAssignmentList_PrimeOutputs(t *testing.T) {

}
