package graphs

import (
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/services"
)

type PrecompDispatchStream struct {
	G               *cyclic.Group
	PublicCypherKey *cyclic.Int
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

var DecryptElgamal = services.Module{
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
}
