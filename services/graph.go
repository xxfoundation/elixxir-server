////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package services

import (
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/internal/measure"
	"math"
	"sync"
	"sync/atomic"
)

const (
	OutputIsBatchsize = InputIsBatchSize
	AutoOutputSize    = AutoInputSize
	AutoNumThreads    = 0
)

type Graph struct {
	generator   GraphGenerator
	modules     map[uint64]*Module
	firstModule *Module
	lastModule  *Module

	name string

	outputModule *Module

	idCount uint64

	stream Stream

	batchSize       uint32
	expandBatchSize uint32

	built  bool
	linked bool

	outputChannel IO_Notify

	sentInputs *uint32

	outputSize      uint32
	outputThreshold float32

	overrideBatchSize uint32

	// NOTE: This mutex is only used for metrics
	sync.Mutex
	metrics measure.Metrics

	errorHandler ErrorCallback
}

// This is too long of a function
func (g *Graph) Build(batchSize uint32, errorHandler ErrorCallback) {

	if g.overrideBatchSize != 0 {
		batchSize = g.overrideBatchSize
	}

	g.errorHandler = errorHandler

	//Checks graph is properly formatted
	err := g.checkGraph()
	if err != nil {
		jww.FATAL.Printf("CheckGraph failed : %+v", err)
	}

	//Find expanded batch size
	var integers []uint32

	for _, m := range g.modules {
		m.checkParameters(g.generator.minInputSize, g.generator.defaultNumTh)
		if m.InputSize != InputIsBatchSize {
			integers = append(integers, m.InputSize)
		}
	}

	integers = append(integers, g.generator.minInputSize)
	integers = append(integers, g.outputSize)
	lcm := globals.LCM(integers)

	expandBatchSize := uint32(math.Ceil(float64(batchSize)/float64(lcm))) * lcm

	g.batchSize = batchSize
	g.expandBatchSize = expandBatchSize

	/*setup output module*/
	g.outputModule = &Module{
		InputSize:      g.outputSize,
		StartThreshold: g.outputThreshold,
		inputModules:   []*Module{g.lastModule},
		Name:           "Output",
		copy:           true,
	}

	g.lastModule.outputModules = append(g.lastModule.outputModules, g.outputModule)
	g.add(g.outputModule)

	/*build assignments*/
	for _, m := range g.modules {
		m.buildAssignments(expandBatchSize)
	}

	g.built = true

	//populate channels
	g.firstModule.open(g.expandBatchSize)
	g.lastModule.open(g.expandBatchSize)

	for _, m := range g.modules {
		if m.id != g.firstModule.id && m.id != g.lastModule.id {
			m.open(0)
		}
	}
	/*finish setting up output*/
	g.outputChannel = g.outputModule.input

	delete(g.modules, g.outputModule.id)
}

// This has to be global to communicate between checkDAG and checkAllNodesUsed
// It has to be lockable, otherwise multiple threads checking the graph at the
// same time can overwrite the mods value and break checkAllNodesUsed()
var visitedModules struct {
	sync.Mutex
	mods []uint64
}

// checkGraph checks that the graph is valid, meaning more than 1 vertex, has a
// first and last module, is a Directed Acyclic Graph, and all vertexes in the
// graph are used
func (g *Graph) checkGraph() error {
	//Check if graph has modules
	if len(g.modules) == 0 {
		return fmt.Errorf("no modules in graph")
	}

	if g.firstModule == nil {
		return fmt.Errorf("no first module")
	}

	if g.lastModule == nil {
		return fmt.Errorf("no last module")
	}

	if g.firstModule == g.lastModule || len(g.modules) == 1 {
		return nil
	}

	// Make an array of visited modules, containing the first ID
	visited := make([]uint64, 0)
	// Clear the visitedModules
	visitedModules.Lock()
	visitedModules.mods = make([]uint64, 0)
	// Start checking based on the firstModule
	err := g.checkDAG(g.firstModule, visited)
	if err != nil {
		return err
	}

	err = g.checkAllNodesUsed()
	visitedModules.Unlock()
	if err != nil {
		return err
	}
	return nil
}

// checkAllNodesUsed checks that all nodes in a graph are called
func (g *Graph) checkAllNodesUsed() error {
	for _, v := range g.modules {
		seen := false
		for _, y := range visitedModules.mods {
			if v.id == y {
				seen = true
			}
		}
		if seen == false {
			return fmt.Errorf("graph vertex %d was not used in graph anywhere", v.id)
		}
	}
	return nil
}

// checkDAG checks that no nodes cause a loopback or are run twice in any path
// A graph loopback occurs when a node tries to call a node already called back
// the chain.
func (g *Graph) checkDAG(mod *Module, visited []uint64) error {
	// Add node to visitedModules, since it's just being visited
	visitedModules.mods = append(visitedModules.mods, mod.id)

	// Reached the end of this path, check that the end is the lastModule
	if len(mod.outputModules) == 0 && mod.id != g.lastModule.id {
		return fmt.Errorf("graph path ended at vertex ID %d,"+
			" not lastModule ID %d", mod.id, g.lastModule.id)
	}

	// Check that this node isn't already in the visited path
	for i, visitedModule := range visited {
		if mod.id == visitedModule {
			return fmt.Errorf("vertex %d was visited multiple times", i)
		}
	}

	// Recurse for all output modules to this one
	for i := range mod.outputModules {
		e := g.checkDAG(mod.outputModules[i], append(visited, mod.id))
		if e != nil {
			return e
		}
	}

	return nil
}

func (g *Graph) Run() {
	if !g.built {
		jww.FATAL.Panicf("graph not built")
	}

	if !g.linked {
		jww.FATAL.Panicf("stream not linked and built")
	}

	for i, m := range g.modules {
		i = i << 8 // high part of int
		for j := uint8(0); j < m.NumThreads; j++ {
			go dispatch(g, m, i+uint64(j))
		}
	}
}

func (g *Graph) Connect(a, b *Module) {

	g.add(a)
	g.add(b)

	a.outputModules = append(a.outputModules, b)
	b.inputModules = append(b.inputModules, a)
}

func (g *Graph) Link(grp *cyclic.Group, source ...interface{}) {
	g.stream.Link(grp, g.expandBatchSize, source...)
	g.linked = true
}

func (g *Graph) First(f *Module) {
	g.add(f)
	g.firstModule = f
}

func (g *Graph) Last(l *Module) {
	g.add(l)
	g.lastModule = l
}

func (g *Graph) add(m *Module) {
	if !m.copy {
		jww.FATAL.Panicf("cannot build a graph with an original module, " +
			"must use a copy")
	}
	m.used = true
	_, ok := g.modules[m.id]

	if !ok {
		g.idCount++
		m.id = g.idCount
		g.modules[m.id] = m
	}
}

func (g *Graph) GetStream() Stream {
	return g.stream
}

func (g *Graph) OverrideBatchSize(b uint32) {
	g.overrideBatchSize = b
}

type Measure func(tag string)

func (g *Graph) Send(chunk Chunk, measureObj Measure) {

	srList, err := g.firstModule.assignmentList.PrimeOutputs(chunk)

	//fmt.Println(g.name,"sending", chunk, "srList", srList)

	if err != nil {
		g.errorHandler(g.name, "input", err)
	}

	for _, r := range srList {
		g.firstModule.input <- r
	}

	//If the entire batch has been sent then send the difference between batchsize and expanded batchsize
	numSent := atomic.AddUint32(g.sentInputs, chunk.Len())

	if numSent == g.batchSize && g.batchSize < g.expandBatchSize {
		endChunk := NewChunk(g.batchSize, g.expandBatchSize)
		srList, err = g.firstModule.assignmentList.PrimeOutputs(endChunk)

		if err != nil {
			g.errorHandler(g.name, "input", err)
		}

		for _, r := range srList {
			g.firstModule.input <- r
		}
	}

	done, err := g.firstModule.assignmentList.DenoteCompleted(len(srList))

	if err != nil {
		g.errorHandler(g.name, "input", err)
	}

	if done {
		//fmt.Println(g.name,"done sending to graph")
		// FIXME: Perhaps not the correct place to close the channel.
		// Ideally, only the sender closes, and only if there's one sender.
		// Does commenting this fix the double close?
		// It does not.
		g.firstModule.closeInput()
		if measureObj != nil {
			measureObj(measure.TagReceiveLastSlot)
		}
	}
}

// outputs from the last op in the graph get sent on this channel.
func (g *Graph) GetOutput() (Chunk, bool) {
	var chunk Chunk
	var ok bool
	for true {
		chunk, ok = <-g.outputChannel
		if chunk.end > g.batchSize {
			if chunk.begin < g.batchSize {
				chunk.end = g.batchSize
			} else {
				continue
			}
		}
		break
	}
	return chunk, ok
}

func (g *Graph) GetExpandedBatchSize() uint32 {
	return g.expandBatchSize
}

func (g *Graph) GetBatchSize() uint32 {
	return g.batchSize
}

func (g *Graph) GetName() string {
	return g.name
}

//Returns all modules with the passed name. used for testing.
func (g *Graph) GetModuleByName(name string) []*Module {
	var moduleList []*Module

	for _, m := range g.modules {
		if m.Name == name {
			moduleList = append(moduleList, m)
		}
	}

	return moduleList
}
