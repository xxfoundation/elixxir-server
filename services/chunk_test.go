package services

import "testing"
//Test that each variable in a chunk is actually uint32
// and what happens if you try to input a number greater than that


func TestNewChunk(t *testing.T) {

	//max value of uint32 is 0 , 4294967295 check it fails if exceeded
	c3 := NewChunk(123,100)

	if( c3.Begin() != 123 && c3.end != 100){
		// TEST IS FAILING
		t.Logf("Chunk does not contain inputed values")
		t.Fail()
	}
}

func TestChunk_Begin(t *testing.T) {
	c3 := NewChunk(174,0)

	if( c3.Begin() != 174){
		// TEST IS FAILING
		t.Logf("Chunk Begin is not correct")
		t.Fail()
	}
}

func TestChunk_End(t *testing.T) {
	c3 := NewChunk(0,2972)

	if( c3.End() != 2972){
		// TEST IS FAILING
		t.Logf("Chunk End is not correct")
		t.Fail()
	}
}

func TestChunk_Len(t *testing.T) {
	c3 := NewChunk(0,2972)

	if( c3.Len() != 2972){
		// TEST IS FAILING
		t.Logf("Chunk length is not equal to expected value")
		t.Fail()

	}
}