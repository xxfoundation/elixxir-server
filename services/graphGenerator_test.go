package services

import (
	"runtime"
	"testing"
)

var GCPanicHandler ErrorCallback = func(g, m string, err error) {
	panic(err)
}


func assertPanic(t *testing.T, f func()) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()
	f()
}

func TestNewGraphGenerator(t *testing.T) {
	//Test defaultNumTH set to 0 fails
	gcTest := func (){
		NewGraphGenerator(4, GCPanicHandler, 0, 1, 0)
	}
	assertPanic(t, gcTest)

	//Test minInputSize = 0 fails
	gcTest = func (){
		NewGraphGenerator(0, GCPanicHandler, 1, 1, 0)
	}
	assertPanic(t, gcTest)

	//Test if outputSize < 0 it fails
	gcTest = func (){
		NewGraphGenerator(1, GCPanicHandler, 1, 1, -1)
	}
	assertPanic(t, gcTest)

	//Test OutputThreshold > 1 it fails
	gcTest = func (){
		NewGraphGenerator(1, GCPanicHandler, 1, 1, 2)
	}
	assertPanic(t, gcTest)

	gc := NewGraphGenerator(1, GCPanicHandler, 1, 1, 0)

	if(gc.defaultNumTh != 1 || gc.minInputSize != 1 || gc.outputSize != 1 || gc.outputThreshold != 0 ){
		t.Logf("Graph Generator returned unexpected value")
		t.Fail()
	}

}

func TestGraphGenerator_NewGraph(t *testing.T) {

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

func TestGraphGenerator_GetErrorHandler(t *testing.T) {

}

func TestGraphGenerator_SetErrorHandler(t *testing.T) {
	//gc := NewGraphGenerator(4, GCPanicHandler, uint8(runtime.NumCPU()), 1, 0)

	// TODO: whats the best way of testing this?
	//var GCPanicHandlerTest ErrorCallback = func(g, m string, err error) {
	//	panic(err)
	//}

}
