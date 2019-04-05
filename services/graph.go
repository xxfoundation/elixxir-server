package services

import (
	"fmt"
	"gitlab.com/elixxir/server/globals"
	"math"
	"time"
)

// Should probably add more params to this like block ID, worker thread ID, etc
type ErrorCallback func(err error)

type Graph struct {
	callback    ErrorCallback
	modules     map[uint64]*Module
	firstModule *Module
	lastModule  *Module

	outputModule *Module

	idCount uint64

	stream Stream

	batchSize       uint32
	expandBatchSize uint32

	built  bool
	linked bool

	outputChannel OutputNotify
}

func NewGraph(callback ErrorCallback) *Graph {
	var g Graph
	g.callback = callback
	g.modules = make(map[uint64]*Module)
	g.idCount = 0
	g.batchSize = 0
	g.expandBatchSize = 0

	g.built = false
	g.linked = false

	return &g
}

// This is too long of a function
func (g *Graph) Build(batchSize uint32, stream Stream)uint32 {

	g.stream = stream

	//Check if graph has modules
	if len(g.modules) == 0 {
		panic("No modules in graph")
	}

	if g.firstModule == nil {
		panic("No first module")
	}

	if g.lastModule == nil {
		panic("No last module")
	}

	//TODO: Check that the graph meets more criteria

	//Find expanded batch size
	integers := make([]uint32, len(g.modules)+1)
	var modulesAtBatchSize []*Module

	for itr, m := range g.modules {
		if m.NumThreads == 0 {
			panic(fmt.Sprintf("Module %s cannot have zero threads", m.Name))
		}

		if m.ChunkSize == 0 {
			if m.AssignmentSize != 0 {
				m.ChunkSize = m.AssignmentSize
			}
		} else {
			if m.ChunkSize < globals.MinSlotSize {
				panic(fmt.Sprintf("ChunkSize (%v) cannot be smaller than the minimum slot range (%v), "+
					"Module: %s", m.ChunkSize, globals.MinSlotSize, m.Name))
			}

			if m.AssignmentSize%m.ChunkSize != 0 {
				panic(fmt.Sprintf("Chunk size (%v) must be a factor or AssignmentSize (%v), "+
					"Module: %s", m.ChunkSize, m.AssignmentSize, m.Name))
			}

			if m.ChunkSize > m.AssignmentSize {
				panic(fmt.Sprintf("ChunkSize (%v) must be <= AssignmentSize (%v), "+
					"Module: %s", m.ChunkSize, m.AssignmentSize, m.Name))
			}
		}

		if m.AssignmentSize == 0 {
			if m.ChunkSize != 0 {
				integers[itr-1] = m.ChunkSize
			}
			modulesAtBatchSize = append(modulesAtBatchSize, m)
		} else {
			integers[itr-1] = m.AssignmentSize
		}

	}

	integers[len(integers)-1] = globals.MinSlotSize
	lcm := globals.LCM(integers)

	expandBatchSize := uint32(math.Ceil(float64(batchSize)/float64(lcm))) * lcm

	g.batchSize = batchSize
	g.expandBatchSize = expandBatchSize

	for _, m := range modulesAtBatchSize {
		m.AssignmentSize = expandBatchSize
		if m.ChunkSize == 0 {
			m.ChunkSize = expandBatchSize
		}
	}

	g.outputModule = &Module{
		AssignmentSize: globals.MinSlotSize,
		ChunkSize:      globals.MinSlotSize,
		inputModules:   []*Module{g.lastModule},
		Name:           "Output",
	}

	g.lastModule.outputModules = append(g.lastModule.outputModules, g.outputModule)
	g.add(g.outputModule)

	/*build assignments*/
	for _, m := range g.modules {

		numJobs := uint32(expandBatchSize / m.AssignmentSize)

		numInputModules := uint32(len(m.inputModules))
		if numInputModules < 1 {
			numInputModules = 1
		}

		jobMaxCount := m.AssignmentSize * numInputModules

		m.assignmentList.assignments = make([]*assignment, numJobs)

		m.assignmentList.size = m.AssignmentSize

		m.assignmentList.completed = new(uint32)

		for j := uint32(0); j < numJobs; j++ {
			m.assignmentList.assignments[j] = newAssignment(uint32(j*m.AssignmentSize), jobMaxCount, m.AssignmentSize, m.ChunkSize)
		}
	}
	g.built = true

	//populate channels
	for _, m := range g.modules {
		m.input = make(OutputNotify, 8)
	}

	g.outputChannel = g.outputModule.input

	delete(g.modules, g.outputModule.id)

	return lcm
}

func (g *Graph) Run() {
	if !g.built {
		panic("graph not built")
	}

	if !g.linked {
		panic("stream not linked and built")
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

func (g *Graph) Link(source ...interface{}) {
	g.stream.Link(g.expandBatchSize, source...)
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

func (g *Graph) Send(sr Chunk) {

	srList, numComplete := g.firstModule.assignmentList.PrimeOutputs(sr)

	for _, r := range srList {
		g.firstModule.input <- r
	}

	done := g.firstModule.assignmentList.DenoteCompleted(numComplete)

	if done {
		// FIXME: Perhaps not the correct place to close the channel.
		// Ideally, only the sender closes, and only if there's one sender.
		// Does commenting this fix the double close?
		// It does not.
		g.firstModule.closeInput()
	}
}

// Outputs from the last op in the graph get sent on this channel.
func (g *Graph) ChunkDoneChannel() OutputNotify {
	return g.outputChannel
}

func (g *Graph) Cap() uint32 {
	return g.expandBatchSize
}

func (g *Graph) Len() uint32 {
	return g.batchSize
}

// This doesn't quite seem robust
func (g *Graph) Kill() bool {
	success := true
	for _, m := range g.modules {
		success = success && m.state.Kill(time.Millisecond*10)
	}
	return success
}
