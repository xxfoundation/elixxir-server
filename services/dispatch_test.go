////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package services

import (
	"fmt"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"math"
	"math/rand"
	"runtime"
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

func (s *Stream1) GetName() string {
	return "Stream1"
}

func (s *Stream1) Link(grp *cyclic.Group, BatchSize uint32, source interface{}) {
	round := source.(*RoundBuffer)
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

func (s *Stream1) Input(index uint32, msg *mixmessages.Slot) error {
	return nil
}
func (s *Stream1) Output(index uint32) *mixmessages.Slot { return nil }

var PanicHandler ErrorCallback = func(err error) {
	panic(err)
}

var ModuleA = Module{
	Adapt: func(streamInput Stream, cryptop cryptops.Cryptop, chunk Chunk) error {
		stream, ok := streamInput.(*Stream1)
		f, ok2 := cryptop.(AddPrototype)

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
		f, ok2 := cryptop.(MultiMulPrototype)

		if !ok || !ok2 {
			return InvalidTypeAssert
		}

		for slot := sRange.Begin(); slot < sRange.End(); slot += f.GetInputSize() {
			regionEnd := slot + f.GetInputSize()
			f(stream.C[slot:regionEnd], stream.D[slot:regionEnd], stream.E[slot:regionEnd])
		}

		return nil
	},
	Cryptop:    MultiMul,
	InputSize:  AUTO_INPUTSIZE,
	NumThreads: 2,
	Name:       "ModuleB",
}

var ModuleC = Module{
	Adapt: func(streamInput Stream, cryptop cryptops.Cryptop, chunk Chunk) error {
		stream, ok := streamInput.(*Stream1)
		f, ok2 := cryptop.(ModMulPrototype)

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
	InputSize:  5,
	Name:       "ModuleC",
}

var ModuleD = Module{
	Adapt: func(streamInput Stream, cryptop cryptops.Cryptop, chunk Chunk) error {
		stream, ok := streamInput.(*Stream1)
		f, ok2 := cryptop.(SubPrototype)

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
	InputSize:  14,
	Name:       "ModuleD",
}

func TestGraph(t *testing.T) {

	grp := initDispatchGroup()

	batchSize := uint32(1000)

	gc := NewGraphGenerator(4, PanicHandler, uint8(runtime.NumCPU()), 1, 0)

	g := gc.NewGraph("test", &Stream1{})

	moduleA := ModuleA.DeepCopy()
	moduleB := ModuleB.DeepCopy()
	moduleC := ModuleC.DeepCopy()
	moduleD := ModuleD.DeepCopy()

	g.First(moduleA)
	g.Connect(moduleA, moduleB)
	g.Connect(moduleB, moduleD)
	g.Connect(moduleA, moduleC)
	g.Connect(moduleC, moduleD)
	g.Last(moduleD)

	g.Build(batchSize)

	roundSize := uint32(math.Ceil(1.2 * float64(g.GetExpandedBatchSize())))
	roundBuf := RoundBuffer{}
	roundBuf.Build(roundSize)

	g.Link(grp, &roundBuf)

	g.Run()

	go func(g *Graph) {

		for i := uint32(0); i < g.GetBatchSize(); i++ {
			g.Send(NewChunk(i, i+1))
		}
	}(g)

	endCh := make(chan bool)

	go func(graph *Graph, encCh chan bool) {

		stream := graph.GetStream().(*Stream1)

		ok := true
		var chunk Chunk

		for ok {
			chunk, ok = graph.GetOutput()
			for i := chunk.Begin(); i < chunk.End(); i++ {
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
		encCh <- true
	}(g, endCh)

	<-endCh
}

type AddPrototype func(X, Y int) int

var Add AddPrototype = func(X, Y int) int {
	return X + Y
}

func (AddPrototype) GetName() string {
	return "Add"
}

func (AddPrototype) GetInputSize() uint32 {
	return 1
}

type ModMulPrototype func(X, Y, P int) int

var ModMul ModMulPrototype = func(X, Y, P int) int {
	return int(math.Abs(float64(X*Y))) % P
}

func (ModMulPrototype) GetName() string {
	return "ModMul"
}

func (ModMulPrototype) GetInputSize() uint32 {
	return 1
}

type MultiMulPrototype func(X, Y, Z []int) []int

var MultiMul MultiMulPrototype = func(X, Y, Z []int) []int {
	for i := range Z {
		Z[i] = X[i] * Y[i]
	}
	return Z
}

func (MultiMulPrototype) GetName() string {
	return "Mul"
}

func (MultiMulPrototype) GetInputSize() uint32 {
	return 4
}

type SubPrototype func(X, Y int) int

var Sub SubPrototype = func(X, Y int) int {
	return X - Y
}

func (SubPrototype) GetName() string {
	return "Sub"
}

func (SubPrototype) GetInputSize() uint32 {
	return 1
}

func initDispatchGroup() *cyclic.Group {
	primeString := "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
		"29024E088A67CC74020BBEA63B139B22514A08798E3404DD" +
		"EF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245" +
		"E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7ED" +
		"EE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3D" +
		"C2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F" +
		"83655D23DCA3AD961C62F356208552BB9ED529077096966D" +
		"670C354E4ABC9804F1746C08CA18217C32905E462E36CE3B" +
		"E39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9" +
		"DE2BCBF6955817183995497CEA956AE515D2261898FA0510" +
		"15728E5A8AACAA68FFFFFFFFFFFFFFFF"
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16), large.NewInt(2), large.NewInt(1283))
	return grp
}
