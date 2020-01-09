package services

import (
	"errors"
	"runtime"
	"testing"
)

var GCPanicHandler ErrorCallback = func(g, m string, err error) {
	panic(err)
}

// We us this to catch panics that we expect so we can test appropriately.
func assertPanic(t *testing.T, f func()) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()
	f()
}

// In this test we test that NewGraph Generator will fail under specific conditions
// and that a newly generated graph will contain the expected values given predetermined inputs
func TestNewGraphGenerator(t *testing.T) {
	//Test defaultNumTH set to 0 fails
	gcTest := func() {
		NewGraphGenerator(4, GCPanicHandler, 0, 1, 0)
	}
	assertPanic(t, gcTest)

	//Test minInputSize = 0 fails
	gcTest = func() {
		NewGraphGenerator(0, GCPanicHandler, 1, 1, 0)
	}
	assertPanic(t, gcTest)

	//Test if outputSize < 0 it fails
	gcTest = func() {
		NewGraphGenerator(1, GCPanicHandler, 1, 1, -1)
	}
	assertPanic(t, gcTest)

	//Test OutputThreshold > 1 it fails
	gcTest = func() {
		NewGraphGenerator(1, GCPanicHandler, 1, 1, 2)
	}
	assertPanic(t, gcTest)

	// Test that graph generator returns a graph with expected values
	gc := NewGraphGenerator(1, GCPanicHandler, 1, 1, 0)
	if gc.defaultNumTh != 1 || gc.minInputSize != 1 || gc.outputSize != 1 || gc.outputThreshold != 0 {
		t.Logf("Graph Generator returned unexpected value")
		t.Fail()
	}

}

func TestGraphGenerator_NewGraph(t *testing.T) {
	stream := &Stream1{}
	gg := NewGraphGenerator(4, GCPanicHandler, 1, 1, 0)
	newGraph := gg.NewGraph("testGraph", stream)

	if newGraph.stream != stream {
		t.Logf("Graphgenerator assigning wrong Stream")
		t.Fail()
	}

	if newGraph.name != "testGraph" {
		t.Logf("GraphGenerator assigning wrong name")
		t.Fail()
	}
}

func TestGraphGenerator_GetDefaultNumTh(t *testing.T) {
	gc := NewGraphGenerator(4, GCPanicHandler, 1, 1, 0)

	if gc.GetDefaultNumTh() != 1 {
		t.Logf("GetDefualtTh returned unexpected value")
		t.Fail()
	}

	gc.defaultNumTh = 2
	if gc.GetDefaultNumTh() != 2 {
		t.Logf("GetDefualtTh returned unexpected value")
		t.Fail()
	}
}

func TestGraphGenerator_GetMinInputSize(t *testing.T) {
	gc := NewGraphGenerator(4, GCPanicHandler, uint8(runtime.NumCPU()), 1, 0)

	if gc.GetMinInputSize() != 4 {
		t.Logf("GetMinInputSize returned unexpected value")
		t.Fail()
	}

	//Change value
	gc.minInputSize = 6
	if gc.GetMinInputSize() != 6 {
		t.Logf("GetMinInputSize returned unexpected value")
		t.Fail()
	}
}

func TestGraphGenerator_GetOutputSize(t *testing.T) {
	gc := NewGraphGenerator(1, GCPanicHandler, uint8(runtime.NumCPU()), 1, 0)

	if gc.GetMinInputSize() != 1 {
		t.Logf("GetOutputSize returned unexpected value")
		t.Fail()
	}

	//Change value
	gc.minInputSize = 2
	if gc.GetMinInputSize() != 2 {
		t.Logf("GetOutputSize returned unexpected value")
		t.Fail()
	}
}

func TestGraphGenerator_GetOutputThreshold(t *testing.T) {
	gc := NewGraphGenerator(4, GCPanicHandler, uint8(runtime.NumCPU()), 1, 0)

	if gc.GetOutputThreshold() != 0 {
		t.Logf("GetOutputThreshold returned unexpected value")
		t.Fail()
	}

	//Change value
	gc.outputThreshold = 2
	if gc.GetOutputThreshold() != 2 {
		t.Logf("GetOutputThreshhold returned unexpected value")
		t.Fail()
	}
}

// Test that GetErrorHandler returns expected
func TestGraphGenerator_GetErrorHandler(t *testing.T) {

	gc := NewGraphGenerator(4, GCPanicHandler, uint8(runtime.NumCPU()), 1, 0)
	var errHandler = gc.GetErrorHandler()
	var f = func() {
		errHandler("", "", errors.New("test error"))
	}

	// Test that the error handler returend is the error handler that panics
	assertPanic(t, f)
}

func TestGraphGenerator_SetErrorHandler(t *testing.T) {
	var nopanicHandler ErrorCallback = func(g, m string, err error) {
		t.Log("The graph generator setHandler is not working, this should have paniced")
		t.Fail()
	}

	gc := NewGraphGenerator(4, nopanicHandler, uint8(runtime.NumCPU()), 1, 0)
	//Change it from a error handler that panics
	gc.SetErrorHandler(GCPanicHandler)

	var errHandler = gc.GetErrorHandler()
	var f = func() {
		errHandler("", "", errors.New("test error"))
	}

	// Test that the error handler is the error handler that panics
	assertPanic(t, f)
}
