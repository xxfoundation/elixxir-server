package services

import "testing"

//Test that Enqueque overflows when weight > maxCount,
func TestAssignment_Enqueue(t *testing.T) {
	a := newAssignment(100)
	//Purposely cause an overflow
	count, err := a.Enqueue(102, 101)
	if count != false {
		t.Logf("Enqueue should have overflowed")
		t.Fail()
	}

	if err == nil {
		//error here is expected so something is wrong
		t.Logf("Enqueue should have overflowed")
		t.Fail()
	}

	//test that an exepected enqueue works
	b := newAssignment(100)
	//Purposely cause an overflow
	count, err = b.Enqueue(100, 101)
	if count != false {
		t.Logf("Enqueue failed, expecting false for maxcount = count")
		t.Fail()
	}

	if err != nil {
		//error here is expected so something is wrong
		t.Logf("Enqueue failed not expecting error %v", err)
		t.Fail()

	}

	//Tests that we can the
	count, err = b.Enqueue(1, 101)

	if count != true {
		t.Logf("Enqueue failed to add weight to assignement.count atomicly")
		t.Fail()
	}

	if err != nil {
		//error here is expected so something is wrong
		t.Logf("Enqueue failed not expecting error %v", err)
		t.Fail()

	}

}

func TestAssignment_GetChunk(t *testing.T) {
	a := newAssignment(100)
	count := a.count
	//Test its usability
	testChunk := a.GetChunk(100)

	if testChunk.begin != 100 {
		t.Logf("GetChunk failed begin should be 100")
		t.Fail()
	}
	if testChunk.end != 200 {
		t.Logf("GetChunk failed begin should be 200")
		t.Fail()
	}

	//test that getchunk doesnt affect the variables in assignment object results should be the same
	if count != a.count {
		t.Logf("GetChunk changed the count value inside the assignment object")
		t.Fail()
	}

	if a.start != 100 {
		t.Logf("GetChunk changed the start values inside the assignment object")
		t.Fail()
	}

}

// Test that DenoteCompleted fails if we try to denote more assignments completed than available,
// also test that it works as expected given expected inputs.
func TestAssignmentList_DenoteCompleted(t *testing.T) {
	testAssignment := newAssignment(100)
	testChunk := NewChunk(0, 100)
	zero := uint32(0)
	al := assignmentList{[]*assignment{testAssignment}, &zero, &zero, &zero, 0, 0, 0, []Chunk{testChunk}}

	// Test completed 1 assignment as loaded
	completed, err := al.DenoteCompleted(1)
	if err == nil && completed != true {
		t.Logf("An unexpected error was thrown")
		t.Fail()
	}

	// Test that error is thrown when we complete more than possible
	completed, err = al.DenoteCompleted(2)
	if err != nil && completed != false {
		t.Logf("Expected error here completed more assignments than possible")
		t.Fail()
	}
}

// Test what we can confirm should be the same,
// and that if a module is in use that it should not be copyable
func TestModule_DeepCopy(t *testing.T) {
	newModuleA := ModuleA.DeepCopy()

	if newModuleA.NumThreads != ModuleA.NumThreads {
		t.Logf("NumThreads dailed to deep copy")
		t.Fail()
	}

	if newModuleA.InputSize != ModuleA.InputSize {
		t.Logf("InputSize Failed to deep copy")
		t.Fail()
	}

	if newModuleA.StartThreshold != ModuleA.StartThreshold {
		t.Logf("StartThreshold Failed to deep copy")
		t.Fail()
	}

	if newModuleA.Name != ModuleA.Name {
		t.Logf("Name Failed to deep copy")
		t.Fail()
	}

	if !newModuleA.copy {
		t.Logf("Deep copy failed to set copy = true")
		t.Fail()
	}

	// set the newModuleA used to true,
	// it should cause a panic when attempting to copy

	copyTest := func() {
		newModuleA.used = true
		newModuleA.DeepCopy()
	}
	assertPanic(t, copyTest)

}
