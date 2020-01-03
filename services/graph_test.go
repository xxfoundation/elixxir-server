package services

import (
	"runtime"
	"testing"
)

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
