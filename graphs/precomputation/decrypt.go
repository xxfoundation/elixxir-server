package graphs

import (
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/node"
	"gitlab.com/elixxir/server/services"
)

type PrecompDispatchStream struct {
	Grp             *cyclic.Group
	PublicCypherKey *cyclic.Int

	R *cyclic.IntBuffer
	U *cyclic.IntBuffer

	Y_R *cyclic.IntBuffer
	Y_U *cyclic.IntBuffer

	KeysMsg   *cyclic.IntBuffer
	CypherMsg *cyclic.IntBuffer
	KeysAD    *cyclic.IntBuffer
	CypherAD  *cyclic.IntBuffer
}

func (s *PrecompDispatchStream) GetStreamName() string {
	return "PrecompDispatchStream"
}

func (s *PrecompDispatchStream) Link(batchSize uint32, source ...interface{}) {
	round := source[0].(*node.RoundBuffer)

	s.Grp = round.Grp
	s.PublicCypherKey = round.CypherPublicKey

	s.R = round.R.GetSubBuffer(0, batchSize)
	s.U = round.U.GetSubBuffer(0, batchSize)
	s.Y_R = round.Y_R.GetSubBuffer(0, batchSize)
	s.Y_U = round.Y_U.GetSubBuffer(0, batchSize)

	s.KeysMsg = s.Grp.NewIntBuffer(batchSize, s.Grp.NewInt(1))
	s.CypherMsg = s.Grp.NewIntBuffer(batchSize, s.Grp.NewInt(1))
	s.KeysAD = s.Grp.NewIntBuffer(batchSize, s.Grp.NewInt(1))
	s.CypherAD = s.Grp.NewIntBuffer(batchSize, s.Grp.NewInt(1))
}

var DecryptElgamal = services.Module{
	Adapt: func(streamInput services.Stream, cryptop cryptops.Cryptop, chunk services.Chunk) error {
		s, ok := streamInput.(*PrecompDispatchStream)
		elgamal, ok2 := cryptop.(cryptops.ElGamalPrototype)

		if !ok || !ok2 {
			return services.InvalidTypeAssert
		}

		for i := chunk.Begin(); i < chunk.End(); i++ {
			//Execute elgamal on the keys for the Message
			elgamal(s.Grp, s.R.Get(i), s.Y_R.Get(i), s.PublicCypherKey, s.KeysMsg.Get(i), s.CypherMsg.Get(i))
			//Execute elgamal on the keys for the Associated Data
			elgamal(s.Grp, s.U.Get(i), s.Y_U.Get(i), s.PublicCypherKey, s.KeysAD.Get(i), s.CypherAD.Get(i))
		}
		return nil
	},
	Cryptop:        cryptops.ElGamal,
	NumThreads:     5,
	AssignmentSize: 1,
	ChunkSize:      1,
	Name:           "DecryptElgamal",
}

func InitPrecompDispatchGraph(batchSize uint32, errorHandler services.ErrorCallback) *services.Graph {
	g := services.NewGraph("PrecomputationDecrypt", errorHandler, &PrecompDispatchStream{})

	g.First(&DecryptElgamal)
	g.Last(&DecryptElgamal)

	return g
}
