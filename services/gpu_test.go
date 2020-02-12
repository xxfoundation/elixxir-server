package services

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/gpumaths"
	"math/rand"
	"runtime"
	"testing"
	"time"
)

// Implement mod exp using GPU kernel
var ModuleExpGPU = Module{
	// Populate Adapt late
	// Populate InputSize late using stream pool max size
	Cryptop:        gpumaths.ExpChunk,
	StartThreshold: 0,
	Name:           "ExpGPU",
	// two threads for two streams
	NumThreads: 2,
}

var ModuleExpCPU = Module{
	// Populate Adapt late
	Cryptop:        cryptops.Exp,
	InputSize:      0,
	StartThreshold: 0,
	Name:           "ExpCPU",
	NumThreads:     uint8(runtime.NumCPU()),
}

// Can I skip round buffer and just use the stream instead?
type ExpTestStream struct {
	g *cyclic.Group
	// Use this stream pool for space to run the GPU code
	streamPool *gpumaths.StreamPool
	length     uint32
	a          *cyclic.IntBuffer
	b          *cyclic.IntBuffer
	ABResult   *cyclic.IntBuffer
	c          *cyclic.IntBuffer
	ABCResult  *cyclic.IntBuffer
	d          *cyclic.IntBuffer
	ABCDResult *cyclic.IntBuffer
}

func (s *ExpTestStream) GetName() string {
	return "ExpTestStream"
}

func (s *ExpTestStream) Link(grp *cyclic.Group, BatchSize uint32, source ...interface{}) {
	// For some reason the group that gets passed to Link is nil, so I just use initDispatchGroup instead of understanding why
	s.g = initDispatchGroup()
	s.length = BatchSize
	// All int buffers should be populated already
}

func (s *ExpTestStream) Input(index uint32, msg *mixmessages.Slot) error {
	return nil
}
func (s *ExpTestStream) Output(index uint32) *mixmessages.Slot { return nil }

// Return an ExpTestStream with all int buffers deepcopied.
// Other fields are not deepcopied, as doing so or not doing so doesn't affect the results
// In addition, streamPool involves GPU resources which aren't trivial to deepcopy.
// This is used to ensure that the results are produced only by the graph that runs with this stream,
// and to ensure that the same inputs produce the same results between GPU and CPU math
func (s *ExpTestStream) DeepCopy() *ExpTestStream {
	result := ExpTestStream{
		g:          s.g,
		streamPool: s.streamPool,
		length:     s.length,
		a:          s.a.DeepCopy(),
		b:          s.b.DeepCopy(),
		ABResult:   s.ABResult.DeepCopy(),
		c:          s.c.DeepCopy(),
		ABCResult:  s.ABCResult.DeepCopy(),
		d:          s.d.DeepCopy(),
		ABCDResult: s.ABCDResult.DeepCopy(),
	}
	return &result
}

// Precondition: ExpTestStream is populated with test data
func runTestGraph(stream *ExpTestStream, moduleA, moduleB, moduleC *Module) {
	gc := NewGraphGenerator(4, PanicHandler, uint8(runtime.NumCPU()), 1, 0)
	// need to do _something_ here???
	// what is it?
	g := gc.NewGraph("test", stream)

	g.First(moduleA)
	g.Connect(moduleA, moduleB)
	g.Connect(moduleB, moduleC)
	g.Last(moduleC)
	g.Build(stream.length)
	// 2k bit ints should work OK with powm odd 4096, as there's enough space in a 4096 bit integer to hold a 2048 bit integer
	g.Link(stream.g, stream.length)
	// Does this block until the graph finishes all slots?
	// Would be cool if it did.
	g.Run()
	g.Send(NewChunk(0, stream.length), nil)
	// Wait on graph to run
	ok := true
	for ok {
		_, ok = g.GetOutput()
	}
}

// Populate an int buffer with random numbers in the cyclic group
func randIntBuffer(g *cyclic.Group, n uint32, rand *rand.Rand) *cyclic.IntBuffer {
	result := g.NewIntBuffer(n, g.NewInt(2))
	for i := uint32(0); i < n; i++ {
		g.SetBytes(result.Get(i), randInGroup(g, rand))
	}
	return result
}

// Generate a random byte slice in the group using an arbitrary RNG
// Not crypto: using a PRNG makes the test run faster
func randInGroup(g *cyclic.Group, rand *rand.Rand) []byte {
	result := make([]byte, len(g.GetPBytes()))
	_, err := rand.Read(result)
	if err != nil {
		// Not production code - just panic, it's easier
		panic(err)
	}
	for !g.BytesInside(result) {
		_, err := rand.Read(result)
		if err != nil {
			panic(err)
		}
	}
	return result
}

// Compare modular exponentiation results with Exp kernel with those doing modular exponentiation on the CPU to show they're both the same
func TestCGC(t *testing.T) {
	// Gold results: Cpu, cpu, cpu
	// Compare to:	 Cpu, gpu, cpu
	// Have I checked whether it works the same with a 2048 bit group? (i.e. do you still get correct results anywhere up to 4096 bits?)
	// I'd assume that you would.
	// I also need to check what happens if there are too many bytes on the CPU side!
	// Fun stuff.

	batchSize := uint32(500)

	streamPool, err := gpumaths.NewStreamPool(2, 2048*int(batchSize))
	if err != nil {
		t.Fatal(err)
	}

	// approximate expanded batch size
	expandBatchSize := 6*batchSize
	rand := rand.New(rand.NewSource(1337))
	g := initDispatchGroup()
	goldExp := ExpTestStream{
		streamPool: streamPool,
		length:     batchSize,
		a:          randIntBuffer(g, expandBatchSize, rand),
		b:          randIntBuffer(g, expandBatchSize, rand),
		ABResult:   g.NewIntBuffer(expandBatchSize, g.NewInt(2)),
		c:          randIntBuffer(g, expandBatchSize, rand),
		ABCResult:  g.NewIntBuffer(expandBatchSize, g.NewInt(2)),
		d:          randIntBuffer(g, expandBatchSize, rand),
		ABCDResult: g.NewIntBuffer(expandBatchSize, g.NewInt(2)),
	}

	// Make a copy to make sure we can get the same results in a different way
	testExp := goldExp.DeepCopy()

	// Make sure there's enough space in the stream to fit the input size
	stream := streamPool.TakeStream()
	streamPool.ReturnStream(stream)
	// MaxSlotsExp should be the same with all streams
	if int(gpumaths.ExpChunk.GetInputSize()) > stream.MaxSlotsExp {
		t.Fatalf("stream too small! has %v slots. make it bigger", stream.MaxSlotsExp)
	}

	// TODO populate Adapt late for these modules
	cpuA := ModuleExpCPU.DeepCopy()
	cpuA.Adapt = func(s Stream, cryptop cryptops.Cryptop, chunk Chunk) error {
		stream, ok := s.(*ExpTestStream)
		if !ok {
			return errors.New("Module A CPU: stream type assert failed")
		}
		for i := chunk.Begin(); i < chunk.End(); i++ {
			cryptops.Exp(stream.g, stream.a.Get(i), stream.b.Get(i), stream.ABResult.Get(i))
		}
		return nil
	}
	cpuB := ModuleExpCPU.DeepCopy()
	cpuB.Adapt = func(s Stream, cryptop cryptops.Cryptop, chunk Chunk) error {
		stream, ok := s.(*ExpTestStream)
		if !ok {
			return errors.New("Module B CPU: stream type assert failed")
		}
		for i := chunk.Begin(); i < chunk.End(); i++ {
			cryptops.Exp(stream.g, stream.ABResult.Get(i), stream.c.Get(i), stream.ABCResult.Get(i))
		}
		return nil
	}
	cpuC := ModuleExpCPU.DeepCopy()
	cpuC.Adapt = func(s Stream, cryptop cryptops.Cryptop, chunk Chunk) error {
		stream, ok := s.(*ExpTestStream)
		if !ok {
			return errors.New("Module C CPU: stream type assert failed")
		}
		for i := chunk.Begin(); i < chunk.End(); i++ {
			cryptops.Exp(stream.g, stream.ABCResult.Get(i), stream.d.Get(i), stream.ABCDResult.Get(i))
		}
		return nil
	}

	ModuleExpGPU.InputSize = uint32(stream.MaxSlotsExp)
	gpuB := ModuleExpGPU.DeepCopy()
	gpuB.Adapt = func(s Stream, c cryptops.Cryptop, chunk Chunk) error {
		stream, ok := s.(*ExpTestStream)
		if !ok {
			return errors.New("Module B GPU: stream type assert failed")
		}

		x := stream.ABResult.GetSubBuffer(chunk.Begin(), chunk.End())
		y := stream.c.GetSubBuffer(chunk.Begin(), chunk.End())
		result := stream.ABCResult.GetSubBuffer(chunk.Begin(), chunk.End())
		_, err := gpumaths.ExpChunk(stream.streamPool, stream.g, x, y, result)
		return err
	}
	// Time test graph runs, just for fun
	start := time.Now()
	runTestGraph(&goldExp, cpuA.DeepCopy(), cpuB.DeepCopy(), cpuC.DeepCopy())
	t.Log(time.Since(start))
	start = time.Now()
	runTestGraph(testExp, cpuA.DeepCopy(), gpuB.DeepCopy(), cpuC.DeepCopy())
	t.Log(time.Since(start))
	// TODO Block on graph execution somehow
	for i := uint32(0); i < batchSize; i++ {
		if goldExp.ABCDResult.Get(i).Cmp(testExp.ABCDResult.Get(i)) != 0 {
			t.Errorf("Results differ at slot %v", i)
		}
	}
}

func TestGGG(t *testing.T) {
	// Gold results: Cpu, cpu, cpu
	// Compare to:   Gpu, gpu, gpu
	batchSize := uint32(500)

	streamPool, err := gpumaths.NewStreamPool(2, 2048*int(batchSize))
	if err != nil {
		t.Fatal(err)
	}

	// approximate expanded batch size
	expandBatchSize := 6*batchSize
	rand := rand.New(rand.NewSource(1337))
	g := initDispatchGroup()
	goldExp := ExpTestStream{
		streamPool: streamPool,
		length:     batchSize,
		a:          randIntBuffer(g, expandBatchSize, rand),
		b:          randIntBuffer(g, expandBatchSize, rand),
		ABResult:   g.NewIntBuffer(expandBatchSize, g.NewInt(2)),
		c:          randIntBuffer(g, expandBatchSize, rand),
		ABCResult:  g.NewIntBuffer(expandBatchSize, g.NewInt(2)),
		d:          randIntBuffer(g, expandBatchSize, rand),
		ABCDResult: g.NewIntBuffer(expandBatchSize, g.NewInt(2)),
	}

	// Make a copy to make sure we can get the same results in a different way
	testExp := goldExp.DeepCopy()

	// Make sure there's enough space in the stream to fit the input size
	stream := streamPool.TakeStream()
	streamPool.ReturnStream(stream)
	// MaxSlotsExp should be the same with all streams
	if int(gpumaths.ExpChunk.GetInputSize()) > stream.MaxSlotsExp {
		t.Fatalf("stream too small! has %v slots. make it bigger", stream.MaxSlotsExp)
	}

	// TODO populate Adapt late for these modules
	cpuA := ModuleExpCPU.DeepCopy()
	cpuA.Adapt = func(s Stream, cryptop cryptops.Cryptop, chunk Chunk) error {
		stream, ok := s.(*ExpTestStream)
		if !ok {
			return errors.New("Module A CPU: stream type assert failed")
		}
		for i := chunk.Begin(); i < chunk.End(); i++ {
			cryptops.Exp(stream.g, stream.a.Get(i), stream.b.Get(i), stream.ABResult.Get(i))
		}
		return nil
	}
	cpuB := ModuleExpCPU.DeepCopy()
	cpuB.Adapt = func(s Stream, cryptop cryptops.Cryptop, chunk Chunk) error {
		stream, ok := s.(*ExpTestStream)
		if !ok {
			return errors.New("Module B CPU: stream type assert failed")
		}
		for i := chunk.Begin(); i < chunk.End(); i++ {
			cryptops.Exp(stream.g, stream.ABResult.Get(i), stream.c.Get(i), stream.ABCResult.Get(i))
		}
		return nil
	}
	cpuC := ModuleExpCPU.DeepCopy()
	cpuC.Adapt = func(s Stream, cryptop cryptops.Cryptop, chunk Chunk) error {
		stream, ok := s.(*ExpTestStream)
		if !ok {
			return errors.New("Module C CPU: stream type assert failed")
		}
		for i := chunk.Begin(); i < chunk.End(); i++ {
			cryptops.Exp(stream.g, stream.ABCResult.Get(i), stream.d.Get(i), stream.ABCDResult.Get(i))
		}
		return nil
	}

	ModuleExpGPU.InputSize = uint32(stream.MaxSlotsExp)
	gpuA := ModuleExpGPU.DeepCopy()
	gpuA.Adapt = func(s Stream, c cryptops.Cryptop, chunk Chunk) error {
		stream, ok := s.(*ExpTestStream)
		if !ok {
			return errors.New("Module B GPU: stream type assert failed")
		}

		x := stream.a.GetSubBuffer(chunk.Begin(), chunk.End())
		y := stream.b.GetSubBuffer(chunk.Begin(), chunk.End())
		result := stream.ABResult.GetSubBuffer(chunk.Begin(), chunk.End())
		_, err := gpumaths.ExpChunk(stream.streamPool, stream.g, x, y, result)
		return err
	}
	gpuB := ModuleExpGPU.DeepCopy()
	gpuB.Adapt = func(s Stream, c cryptops.Cryptop, chunk Chunk) error {
		stream, ok := s.(*ExpTestStream)
		if !ok {
			return errors.New("Module B GPU: stream type assert failed")
		}

		x := stream.ABResult.GetSubBuffer(chunk.Begin(), chunk.End())
		y := stream.c.GetSubBuffer(chunk.Begin(), chunk.End())
		result := stream.ABCResult.GetSubBuffer(chunk.Begin(), chunk.End())
		_, err := gpumaths.ExpChunk(stream.streamPool, stream.g, x, y, result)
		return err
	}
	gpuC := ModuleExpGPU.DeepCopy()
	gpuC.Adapt = func(s Stream, c cryptops.Cryptop, chunk Chunk) error {
		stream, ok := s.(*ExpTestStream)
		if !ok {
			return errors.New("Module B GPU: stream type assert failed")
		}

		x := stream.ABCResult.GetSubBuffer(chunk.Begin(), chunk.End())
		y := stream.d.GetSubBuffer(chunk.Begin(), chunk.End())
		result := stream.ABCDResult.GetSubBuffer(chunk.Begin(), chunk.End())
		_, err := gpumaths.ExpChunk(stream.streamPool, stream.g, x, y, result)
		return err
	}
	// Time test graph runs, just for fun
	start := time.Now()
	runTestGraph(&goldExp, cpuA.DeepCopy(), cpuB.DeepCopy(), cpuC.DeepCopy())
	t.Log(time.Since(start))
	start = time.Now()
	runTestGraph(testExp, gpuA.DeepCopy(), gpuB.DeepCopy(), gpuC.DeepCopy())
	t.Log(time.Since(start))
	// TODO Block on graph execution somehow
	for i := uint32(0); i < batchSize; i++ {
		if goldExp.ABCDResult.Get(i).Cmp(testExp.ABCDResult.Get(i)) != 0 {
			t.Errorf("Results differ at slot %v", i)
		}
	}
}
