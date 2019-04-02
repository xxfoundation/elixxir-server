package services

import (
	"fmt"
	"gitlab.com/elixxir/crypto/cryptops"
	"math"
	"math/rand"
	"testing"
)

type RoundBuffer struct {
	A []int
	B []int
	D []int
	F []int
	G []int
}

func (round *RoundBuffer) Build(size uint32) {
	round.A = make([]int, size)
	round.B = make([]int, size)
	round.D = make([]int, size)
	round.F = make([]int, size)
	round.G = make([]int, size)

	src := rand.NewSource(42)
	rng := rand.New(src)

	for i := uint32(0); i < size; i++ {
		round.A[i] = rng.Int() % 107
		round.B[i] = rng.Int() % 107
		round.D[i] = rng.Int() % 107
		round.F[i] = rng.Int() % 107
		round.G[i] = rng.Int() % 107
	}
}

type Stream1 struct {
	Prime int
	A     []int
	B     []int
	C     []int
	D     []int
	E     []int
	F     []int
	G     []int
	H     []int
	I     []int
}

func (s *Stream1) GetStreamName() string {
	return "Stream1"
}

func (s *Stream1) Link(BatchSize uint32, source ...interface{}) {
	round := source[0].(*RoundBuffer)
	s.Prime = 107
	s.A = round.A[:BatchSize]
	s.B = round.B[:BatchSize]
	s.C = make([]int, BatchSize)
	s.D = round.D[:BatchSize]
	s.E = make([]int, BatchSize)
	s.F = round.F[:BatchSize]
	s.G = round.G[:BatchSize]
	s.H = make([]int, BatchSize)
	s.I = make([]int, BatchSize)
}

var PanicHandler ErrorCallback = func(err error) {
	panic(err)
}

var ModuleA = Module{
	Adapt: func(streamInput Stream, cryptop cryptops.Cryptop, chunk Chunk) error {
		stream, ok := streamInput.(*Stream1)
		f, ok2 := cryptop.(AddSignature)

		if !ok || !ok2 {
			return InvalidTypeAssert
		}

		for slot := chunk.Begin(); slot < chunk.End(); slot++ {
			stream.C[slot] = f(stream.A[slot], stream.B[slot])
		}
		return nil
	},
	Cryptop:    Add,
	InputSize:  8,
	NumThreads: 8,
	Name:       "ModuleA",
}

var ModuleB = Module{
	Adapt: func(streamInput Stream, cryptop cryptops.Cryptop, sRange Chunk) error {
		stream, ok := streamInput.(*Stream1)
		f, ok2 := cryptop.(MultiMulSignature)

		if !ok || !ok2 {
			return InvalidTypeAssert
		}

		for slot := sRange.Begin(); slot < sRange.End(); slot += f.GetMinSize() {
			regionEnd := slot + f.GetMinSize()
			f(stream.C[slot:regionEnd], stream.D[slot:regionEnd], stream.E[slot:regionEnd])
		}

		return nil
	},
	Cryptop:    MultiMul,
	InputSize:  MultiMul.GetMinSize(),
	NumThreads: 2,
	Name:       "ModuleB",
}

var ModuleC = Module{
	Adapt: func(streamInput Stream, cryptop cryptops.Cryptop, chunk Chunk) error {
		stream, ok := streamInput.(*Stream1)
		f, ok2 := cryptop.(ModMulSignature)

		if !ok || !ok2 {
			return InvalidTypeAssert
		}

		for slot := chunk.Begin(); slot < chunk.End(); slot++ {
			stream.H[slot] = f(stream.F[slot], stream.G[slot], stream.Prime)
		}

		return nil
	},
	Cryptop:    ModMul,
	NumThreads: 3,
	InputSize:  2,
	Name:       "ModuleC",
}

var ModuleD = Module{
	Adapt: func(streamInput Stream, cryptop cryptops.Cryptop, chunk Chunk) error {
		stream, ok := streamInput.(*Stream1)
		f, ok2 := cryptop.(SubSignature)

		if !ok || !ok2 {
			return InvalidTypeAssert
		}

		for slot := chunk.Begin(); slot < chunk.End(); slot++ {
			stream.I[slot] = f(stream.E[slot], stream.H[slot])
		}
		return nil
	},
	Cryptop:    Sub,
	NumThreads: 5,
	InputSize:  7,
	Name:       "ModuleD",
}

func TestGraph(t *testing.T) {

	batchSize := uint32(1000)

	g := NewGraph(PanicHandler, 0, 0)

	g.First(&ModuleA)
	g.Connect(&ModuleA, &ModuleB)
	g.Connect(&ModuleB, &ModuleD)
	g.Connect(&ModuleA, &ModuleC)
	g.Connect(&ModuleC, &ModuleD)
	g.Last(&ModuleD)

	g.Build(batchSize, &Stream1{})

	roundSize := uint32(math.Ceil(1.2 * float64(g.Cap())))
	roundBuf := RoundBuffer{}
	roundBuf.Build(roundSize)

	g.Link(&roundBuf)

	g.Run()

	go func(g *Graph) {

		for i := uint32(0); i < g.Cap(); i++ {
			g.Send(NewChunk(i, i+1))
		}
	}(g)

	endCh := make(chan bool)

	go func(g *Graph, encCh chan bool) {

		stream := g.GetStream().(*Stream1)

		// This is probably the problem - we're ranging over a channel that
		// doesn't get closed properly. So the range won't finish.
		for lot := range g.LotDoneChannel() {
			for i := lot.Begin(); i < lot.End(); i++ {
				// Compute expected result for this slot
				A := stream.A[i]
				B := stream.B[i]
				C := A + B
				D := stream.D[i]
				E := C * D
				F := stream.F[i]
				G := stream.G[i]
				H := int(math.Abs(float64(F*G))) % stream.Prime
				I := E - H

				if I != stream.I[i] {
					t.Error(fmt.Sprintf("streams not equal on slot %v", i))
				}
			}
		}
		// Did he mean to have encCh and endCh? encCh is probably just a typo
		// Also, is this the send that panics?
		// There's a send on a closed channel that panics every time main is
		// run, so we need to figure out if it's because of something main did,
		// or if it's a problem in the dispatcher.
		encCh <- true
	}(g, endCh)

	<-endCh
}

type AddSignature func(X, Y int) int

var Add AddSignature = func(X, Y int) int {
	return X + Y
}

func (AddSignature) GetName() string {
	return "Add"
}

func (AddSignature) GetMinSize() uint32 {
	return 1
}

type ModMulSignature func(X, Y, P int) int

var ModMul ModMulSignature = func(X, Y, P int) int {
	return int(math.Abs(float64(X*Y))) % P
}

func (ModMulSignature) GetName() string {
	return "ModMul"
}

func (ModMulSignature) GetMinSize() uint32 {
	return 1
}

type MultiMulSignature func(X, Y, Z []int) []int

var MultiMul MultiMulSignature = func(X, Y, Z []int) []int {
	for i := range Z {
		Z[i] = X[i] * Y[i]
	}
	return Z
}

func (MultiMulSignature) GetName() string {
	return "Mul"
}

func (MultiMulSignature) GetMinSize() uint32 {
	return 4
}

type SubSignature func(X, Y int) int

var Sub SubSignature = func(X, Y int) int {
	return X - Y
}

func (SubSignature) GetName() string {
	return "Sub"
}

func (SubSignature) GetMinSize() uint32 {
	return 1
}
