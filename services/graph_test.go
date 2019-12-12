package services

import (
	"runtime"
	"testing"
)

func TestGraph_Build(t *testing.T) {
	//	if len(g.modules) == 0 {
	//		return fmt.Errorf("no modules in graph")
	//	}
	//
	//	if g.firstModule == nil {
	//		return fmt.Errorf("no first module")
	//	}
	//
	//	if g.lastModule == nil {
	//		return fmt.Errorf("no last module")
	//	}


	//Test CheckDag?

	//Test checkALlNodes Used?

}

func TestGraph_Run(t *testing.T) {

}
func TestGraph_Connect(t *testing.T) {

}

func TestGraph_First(t *testing.T) {

}

func TestGraph_Last(t *testing.T) {

}

func TestGraph_GetBatchSize(t *testing.T) {

}

func TestGraph_GetStream(t *testing.T) {
	stream := &Stream1{}
	gc := NewGraphGenerator(4, PanicHandler, uint8(runtime.NumCPU()), 1, 0)
	g := gc.NewGraph("test", stream)

	if g.GetStream() != stream {
		t.Logf("Graph.GetStream returned the wrong stream")
		t.Fail()
	}
}

func TestGraph_GetExpandedBatchSize(t *testing.T) {
	stream := &Stream1{}
	gc := NewGraphGenerator(4, PanicHandler, uint8(runtime.NumCPU()), 1, 0)
	g := gc.NewGraph("test", stream)

	if g.GetStream() != stream {
		t.Logf("Graph.GetStream returned the wrong stream")
		t.Fail()
	}
}

func TestGraph_GetModuleByName(t *testing.T) {
	//What are the modules
}

func TestGraph_GetOutput(t *testing.T) {
	//stream := &Stream1{}
	//gc := NewGraphGenerator(4, PanicHandler, uint8(runtime.NumCPU()), 1, 0)
	//g := gc.NewGraph("test", stream)

	//FIXME: best way of testing this?

}

func TestGraph_GetName(t *testing.T) {
	stream := &Stream1{}
	name := "test123"
	gc := NewGraphGenerator(4, PanicHandler, uint8(runtime.NumCPU()), 1, 0)
	g := gc.NewGraph(name, stream)

	if g.GetName() != name {
		t.Logf("Graph.GetName returned the wrong name")
		t.Fail()
	}
}
