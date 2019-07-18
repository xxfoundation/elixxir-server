////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package services

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/server/phase"
	"math"
	"sync/atomic"
	"time"
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
}

// This is too long of a function
func (g *Graph) Build(batchSize uint32) {

	if g.overrideBatchSize != 0 {
		batchSize = g.overrideBatchSize
	}

	//Checks graph is properly formatted
	g.checkGraph()

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

func (g *Graph) checkGraph() {
	//Check if graph has modules
	if len(g.modules) == 0 {
		jww.FATAL.Panicf("No modules in graph")
	}

	if g.firstModule == nil {
		jww.FATAL.Panicf("No first module")
	}

	if g.lastModule == nil {
		jww.FATAL.Panicf("No last module")
	}
}

func (g *Graph) Run() {
	if !g.built {
		jww.FATAL.Panicf("graph not built")
	}

	if !g.linked {
		jww.FATAL.Panicf("stream not linked and built")
	}

	for _, m := range g.modules {

		m.state.numTh = uint8(m.NumThreads)
		m.state.Init()

		for i := uint8(0); i < m.state.numTh; i++ {
			go dispatch(g, m, uint8(i))
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

func (g *Graph) Send(chunk Chunk, measure phase.Measure) {

	srList, err := g.firstModule.assignmentList.PrimeOutputs(chunk)

	//fmt.Println(g.name,"sending", chunk, "srList", srList)

	if err != nil {
		g.generator.errorHandler(g.name, "input", err)
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
			g.generator.errorHandler(g.name, "input", err)
		}

		for _, r := range srList {
			g.firstModule.input <- r
		}
	}

	done, err := g.firstModule.assignmentList.DenoteCompleted(len(srList))

	if err != nil {
		g.generator.errorHandler(g.name, "input", err)
	}

	if done {
		//fmt.Println(g.name,"done sending to graph")
		// FIXME: Perhaps not the correct place to close the channel.
		// Ideally, only the sender closes, and only if there's one sender.
		// Does commenting this fix the double close?
		// It does not.
		g.firstModule.closeInput()
		if measure != nil {
			measure("Signaling that the last slot has been sent")
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

// This doesn't quite seem robust
func (g *Graph) Kill() bool {
	success := true
	for _, m := range g.modules {
		success = success && m.state.Kill(time.Millisecond*10)
	}
	return success
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
