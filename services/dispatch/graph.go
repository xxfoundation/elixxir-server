package dispatch

import (
	"fmt"
	"gitlab.com/elixxir/dispatch/globals"
	"math"
	"time"
)

type Graph struct {
	callback    ErrorCallback
	modules     map[uint64]*Module
	firstModule *Module
	lastModule  *Module

	outputModule *Module

	idCount uint64

	stream Stream

	batchSize uint32
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
func (g *Graph) Build(batchSize uint32, stream Stream) {

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

	for itr, m := range g.modules {
		if m.NumThreads == 0 {
			panic(fmt.Sprintf("Module %s cannot have zero threads", m.Name))
		}
		if m.InputSize == 0 {
			m.InputSize = m.F.GetMinSize()
		}
		integers[itr-1] = m.InputSize
	}

	integers[len(integers)-1] = globals.MinSlotRange
	lcm := globals.LCM(integers)

	fmt.Printf("LCM: %v\n", lcm)

	expandBatchSize := uint32(math.Ceil(float64(batchSize)/float64(lcm))) * lcm

	g.batchSize = batchSize
	g.expandBatchSize = expandBatchSize

	g.outputModule = &Module{
		InputSize:    globals.MinSlotRange,
		inputModules: []*Module{g.lastModule},
		Name:         "Output",
	}

	g.lastModule.outputModules = append(g.lastModule.outputModules, g.outputModule)
	g.add(g.outputModule)

	/*build assignments*/
	for _, m := range g.modules {
		inputSize := m.InputSize

		if inputSize < globals.MinSlotRange {
			inputSize = globals.MinSlotRange
		}

		numJobs := uint32(expandBatchSize / inputSize)

		numInputModules := uint32(len(m.inputModules))
		if numInputModules < 1 {
			numInputModules = 1
		}

		jobMaxCount := inputSize * numInputModules

		m.assignments = make([]*assignment, numJobs)

		m.assignmentSize = inputSize

		m.assignmentsCompleted = new(uint32)

		for j := uint32(0); j < numJobs; j++ {
			m.assignments[j] = newAssignment(uint32(j*inputSize), jobMaxCount, inputSize)
		}
	}
	g.built = true

	//populate channels
	for _, m := range g.modules {
		m.input = make(OutputNotify)
	}

	g.outputChannel = g.outputModule.input

	delete(g.modules, g.outputModule.id)
}

func (g *Graph) Run() {
	if !g.built {
		panic("graph not built")
	}

	if !g.linked {
		panic("stream not linked and built")
	}

	for _, m := range g.modules {

		m.numTh = uint8(m.NumThreads)
		m.moduleState.Init()

		for i := uint8(0); i < m.numTh; i++ {
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

func (g *Graph) Send(sr Lot) {

	srList, done := g.firstModule.assignmentList.PrimeOutputs(sr)

	for _, r := range srList {
		g.firstModule.input <- r
	}
	if done {
		// FIXME: Perhaps not the correct place to close the channel.
		// Ideally, only the sender closes, and only if there's one sender.
		// Does commenting this fix the double close?
		// It does not.
		close(g.firstModule.input)
	}
}

// Outputs from the last op in the graph get sent on this channel.
func (g *Graph) LotDoneChannel() OutputNotify {
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
		success = success && m.Kill(time.Millisecond*10)
	}
	return success
}
