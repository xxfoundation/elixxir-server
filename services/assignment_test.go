package services

import "testing"

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
