package services

import (
	"runtime"
	"testing"
)

func newGraphAndGeneratorTestUtil() (*Graph, GraphGenerator) {
	stream := &Stream1{}
	name := "test123"
	gc := NewGraphGenerator(4, uint8(runtime.NumCPU()), 1, 0)
	g := gc.NewGraph(name, stream)
	return g, gc
}

func TestGraph_GetStream(t *testing.T) {
	stream := &Stream1{}
	gc := NewGraphGenerator(4, uint8(runtime.NumCPU()), 1, 0)
	g := gc.NewGraph("test", stream)

	if g.GetStream() != stream {
		t.Logf("Graph.GetStream returned the wrong stream")
		t.Fail()
	}
}

// test that we getExpandBatchSize we are expecting
func TestGraph_GetExpandedBatchSize(t *testing.T) {
	g, _ := newGraphAndGeneratorTestUtil()

	if g.GetExpandedBatchSize() != uint32(0) {
		t.Logf("Graph.GetStream returned the wrong stream")
		t.Fail()
	}
}

// Test to make sure GetName is  returning expected name valeue
func TestGraph_GetName(t *testing.T) {
	g, _ := newGraphAndGeneratorTestUtil()

	if g.GetName() != "test123" {
		t.Logf("Graph.GetName returned the wrong name")
		t.Fail()
	}
}

// Test that using graph.first() add the module to the firstModule value
func TestGraph_First(t *testing.T) {
	g, _ := newGraphAndGeneratorTestUtil()
	//Create a module and add it as the first module
	// We deep copy the module because add wont allow us to add an original module
	newModuleA := ModuleA.DeepCopy()
	g.First(newModuleA)

	// Check to see that it appropriately added
	if g.firstModule != newModuleA {
		t.Logf("Graph First is failing to add module as firstModule")
		t.Fail()
	}

	//Now test that when we change the first module it changes appropriately
	newModuleB := ModuleB.DeepCopy()
	g.First(newModuleB)

	// Check to see that it appropriately added
	if g.firstModule != newModuleB {
		t.Logf("Graph First is failing to change module set as firstModule")
		t.Fail()
	}
}

// Test that using graph.Last() adds the module the the lastModule value
func TestGraph_Last(t *testing.T) {
	g, _ := newGraphAndGeneratorTestUtil()
	//Create a module and add it as the first module
	// We deep copy the module because add wont allow us to add an original module
	newModuleA := ModuleA.DeepCopy()
	g.Last(newModuleA)

	// Check to see that it appropriately added
	if g.lastModule != newModuleA {
		t.Logf("Graph First is failing to add module as firstModule")
		t.Fail()
	}

	//Now test that when we change the first module it changes appropriately
	newModuleB := ModuleB.DeepCopy()
	g.Last(newModuleB)

	// Check to see that it appropriately added
	if g.lastModule != newModuleB {
		t.Logf("Graph First is failing to change module set as firstModule")
		t.Fail()
	}
}
