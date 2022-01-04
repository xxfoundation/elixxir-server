///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// +build linux,gpu,cgo

package services

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/gpumathsgo"
	"gitlab.com/elixxir/gpumathsgo/cryptops"
	"math/rand"
	"runtime"
	"testing"
	"time"
)

var (
	// Use GPU to calculate a^b
	gpuA = Module{
		// Cryptop is unused. Unsure if correct
		Adapt: func(s Stream, _ cryptops.Cryptop, chunk Chunk) error {
			stream, ok := s.(*ExpTestStream)
			if !ok {
				return errors.New("Module A GPU: stream type assert failed")
			}

			x := stream.a.GetSubBuffer(chunk.Begin(), chunk.End())
			y := stream.b.GetSubBuffer(chunk.Begin(), chunk.End())
			result := stream.ABResult.GetSubBuffer(chunk.Begin(), chunk.End())
			_, err := gpumaths.ExpChunk(stream.streamPool, stream.g, x, y, result)
			return err
		},
		InputSize:      8,
		Cryptop:        gpumaths.ExpChunk,
		StartThreshold: 0,
		Name:           "ExpGPUA",
		// two threads for two streams
		NumThreads: 2,
	}

	// Use GPU to calculate (a^b)^c
	gpuB = Module{
		// Cryptop is unused. Unsure if correct
		Adapt: func(s Stream, _ cryptops.Cryptop, chunk Chunk) error {
			stream, ok := s.(*ExpTestStream)
			if !ok {
				return errors.New("Module B GPU: stream type assert failed")
			}

			x := stream.ABResult.GetSubBuffer(chunk.Begin(), chunk.End())
			y := stream.c.GetSubBuffer(chunk.Begin(), chunk.End())
			result := stream.ABCResult.GetSubBuffer(chunk.Begin(), chunk.End())
			_, err := gpumaths.ExpChunk(stream.streamPool, stream.g, x, y, result)
			return err
		},
		InputSize:      8,
		Cryptop:        gpumaths.ExpChunk,
		StartThreshold: 0,
		Name:           "ExpGPUB",
		// two threads for two streams
		NumThreads: 2,
	}

	// Use GPU to calculate ((a^b)^c)^d
	gpuC = Module{
		// Cryptop is unused. Unsure if correct
		Adapt: func(s Stream, _ cryptops.Cryptop, chunk Chunk) error {
			stream, ok := s.(*ExpTestStream)
			if !ok {
				return errors.New("Module C GPU: stream type assert failed")
			}

			x := stream.ABCResult.GetSubBuffer(chunk.Begin(), chunk.End())
			y := stream.d.GetSubBuffer(chunk.Begin(), chunk.End())
			result := stream.ABCDResult.GetSubBuffer(chunk.Begin(), chunk.End())
			_, err := gpumaths.ExpChunk(stream.streamPool, stream.g, x, y, result)
			return err
		},
		InputSize:      8,
		Cryptop:        gpumaths.ExpChunk,
		StartThreshold: 0,
		Name:           "ExpGPUC",
		// two threads for two streams
		NumThreads: 2,
	}

	// Use CPU to calculate a^b
	cpuA = Module{
		Adapt: func(s Stream, cryptop cryptops.Cryptop, chunk Chunk) error {
			stream, ok := s.(*ExpTestStream)
			if !ok {
				return errors.New("Module A CPU: stream type assert failed")
			}
			for i := chunk.Begin(); i < chunk.End(); i++ {
				cryptops.Exp(stream.g, stream.a.Get(i), stream.b.Get(i), stream.ABResult.Get(i))
			}
			return nil
		},
		Cryptop:        cryptops.Exp,
		InputSize:      0,
		StartThreshold: 0,
		Name:           "ExpCPU",
		NumThreads:     uint8(runtime.NumCPU()),
	}

	// Use CPU to calculate (a^b)^c
	cpuB = Module{
		Adapt: func(s Stream, cryptop cryptops.Cryptop, chunk Chunk) error {
			stream, ok := s.(*ExpTestStream)
			if !ok {
				return errors.New("Module B CPU: stream type assert failed")
			}
			for i := chunk.Begin(); i < chunk.End(); i++ {
				cryptops.Exp(stream.g, stream.ABResult.Get(i), stream.c.Get(i), stream.ABCResult.Get(i))
			}
			return nil
		},
		Cryptop:        cryptops.Exp,
		InputSize:      0,
		StartThreshold: 0,
		Name:           "ExpCPU",
		NumThreads:     uint8(runtime.NumCPU()),
	}

	// Use CPU to calculate ((a^b)^c)^d
	cpuC = Module{
		Adapt: func(s Stream, cryptop cryptops.Cryptop, chunk Chunk) error {
			stream, ok := s.(*ExpTestStream)
			if !ok {
				return errors.New("Module C CPU: stream type assert failed")
			}
			for i := chunk.Begin(); i < chunk.End(); i++ {
				cryptops.Exp(stream.g, stream.ABCResult.Get(i), stream.d.Get(i), stream.ABCDResult.Get(i))
			}
			return nil
		},
		Cryptop:        cryptops.Exp,
		InputSize:      0,
		StartThreshold: 0,
		Name:           "ExpCPU",
		NumThreads:     uint8(runtime.NumCPU()),
	}
)

// This stream has all the variables and results needed to do exponentiation
// It also includes a stream pool which is needed to run GPU kernels
type ExpTestStream struct {
	g          *cyclic.Group
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

// Implement stream interface
// Returns stream tag to help logging
func (s *ExpTestStream) GetName() string {
	return "ExpTestStream"
}

// Implement stream interface
// Gives stream a group and length
// Int buffers should be populated elsewhere
func (s *ExpTestStream) Link(grp *cyclic.Group, BatchSize uint32, source ...interface{}) {
	// For some reason the group that gets passed to Link is nil, so I just use initDispatchGroup instead of figuring out why
	s.g = initDispatchGroup()
	s.length = BatchSize
}

// Implement stream interface
func (s *ExpTestStream) Input(index uint32, msg *mixmessages.Slot) error { return nil }

// Implement stream interface
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
// Prepares and runs a graph with the specified 3 modules
func runTestGraph(stream *ExpTestStream, moduleA, moduleB, moduleC *Module) {
	gc := NewGraphGenerator(4, uint8(runtime.NumCPU()), 1, 0)
	// need to do _something_ here???
	// what is it?
	g := gc.NewGraph("test", stream)

	g.First(moduleA)
	g.Connect(moduleA, moduleB)
	g.Connect(moduleB, moduleC)
	g.Last(moduleC)
	g.Build(stream.length, PanicHandler)
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
// Shows that the GPU code can work with and integrate with CPU modules if needed
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
	expandBatchSize := 6 * batchSize
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

	// Copy to set custom input size
	gpuBLocal := gpuB.DeepCopy()
	// We could also set input size to be the number of slots that's less than MaxSlotsExp that has a common factor with the CPU input sizes
	// That would allow for fairly large chunks while not exceeding input size
	// The user is responsible for choosing an input size that will work well with both the dispatcher and CUDA
	gpuBLocal.InputSize = 512
	// Time test graph runs, just for fun
	start := time.Now()
	runTestGraph(&goldExp, cpuA.DeepCopy(), cpuB.DeepCopy(), cpuC.DeepCopy())
	t.Log(time.Since(start))
	start = time.Now()
	runTestGraph(testExp, cpuA.DeepCopy(), gpuBLocal.DeepCopy(), cpuC.DeepCopy())
	t.Log(time.Since(start))
	// TODO Block on graph execution somehow
	for i := uint32(0); i < batchSize; i++ {
		if goldExp.ABCDResult.Get(i).Cmp(testExp.ABCDResult.Get(i)) != 0 {
			t.Errorf("Results differ at slot %v", i)
		}
	}
}

// Compare modular exponentiation results with Exp kernel with those doing modular exponentiation on the CPU to show they're both the same
// Shows that the GPU code can work correctly on its own
func TestGGG(t *testing.T) {
	// Gold results: Cpu, cpu, cpu
	// Compare to:   Gpu, gpu, gpu
	batchSize := uint32(500)

	streamPool, err := gpumaths.NewStreamPool(2, 2048*int(batchSize))
	if err != nil {
		t.Fatal(err)
	}

	// approximate expanded batch size
	expandBatchSize := 6 * batchSize
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

	gpuALocal := gpuA.DeepCopy()
	gpuBLocal := gpuB.DeepCopy()
	gpuCLocal := gpuC.DeepCopy()
	gpuALocal.InputSize = 512
	gpuBLocal.InputSize = 512
	gpuCLocal.InputSize = 512

	// Time test graph runs, just for fun
	start := time.Now()
	runTestGraph(&goldExp, cpuA.DeepCopy(), cpuB.DeepCopy(), cpuC.DeepCopy())
	t.Log(time.Since(start))
	start = time.Now()
	runTestGraph(testExp, gpuALocal.DeepCopy(), gpuBLocal.DeepCopy(), gpuCLocal.DeepCopy())
	t.Log(time.Since(start))
	// TODO Block on graph execution somehow
	for i := uint32(0); i < batchSize; i++ {
		if goldExp.ABCDResult.Get(i).Cmp(testExp.ABCDResult.Get(i)) != 0 {
			t.Errorf("Results differ at slot %v", i)
		}
	}
}
