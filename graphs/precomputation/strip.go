package precomputation

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/node"
	"gitlab.com/elixxir/server/services"
)

// This file implements the Graph for the Precomputation Strip phase.
// Strip phase inverts the Round Private Keys and removes the homomorphic encryption
// from the encrypted message keys and encrypted associated data keys, revealing completed precomputation

// Stream holding data containing private key from encrypt and inputs used by strip
type StripStream struct {
	Grp *cyclic.Group

	// Link to round object
	MessagePrecomputation *cyclic.IntBuffer
	ADPrecomputation      *cyclic.IntBuffer

	// Unique to stream
	CypherMsg *cyclic.IntBuffer
	CypherAD  *cyclic.IntBuffer
}

func (s *StripStream) GetName() string {
	return "PrecompStripStream"
}

func (s *StripStream) Link(batchSize uint32, source interface{}) {
	round := source.(*node.RoundBuffer)

	s.Grp = round.Grp

	s.MessagePrecomputation = round.MessagePrecomputation.GetSubBuffer(0, batchSize)
	s.ADPrecomputation = round.ADPrecomputation.GetSubBuffer(0, batchSize)

	s.CypherMsg = s.Grp.NewIntBuffer(batchSize, s.Grp.NewInt(1))
	s.CypherAD = s.Grp.NewIntBuffer(batchSize, s.Grp.NewInt(1))
}

func (s *StripStream) Input(index uint32, slot *mixmessages.CmixSlot) error {

	if index >= uint32(s.CypherMsg.Len()) {
		return node.ErrOutsideOfBatch
	}

	if !s.Grp.BytesInside(slot.PartialMessageCypherText, slot.PartialAssociatedDataCypherText) {
		return node.ErrOutsideOfGroup
	}

	s.Grp.SetBytes(s.CypherMsg.Get(index), slot.PartialMessageCypherText)
	s.Grp.SetBytes(s.CypherAD.Get(index), slot.PartialAssociatedDataCypherText)

	return nil
}

func (s *StripStream) Output(index uint32) *mixmessages.CmixSlot {

	return &mixmessages.CmixSlot{
		PartialMessageCypherText:        s.CypherMsg.Get(index).Bytes(),
		PartialAssociatedDataCypherText: s.CypherAD.Get(index).Bytes(),
	}

}

// Module in precomputation strip implementing cryptops.Inverse
var StripInverse = services.Module{
	// Runs root coprime for cypher message and cypher associated data
	Adapt: func(streamInput services.Stream, cryptop cryptops.Cryptop, chunk services.Chunk) error {
		s, ok := streamInput.(*StripStream)
		inverse, ok2 := cryptop.(cryptops.InversePrototype)

		if !ok || !ok2 {
			return services.InvalidTypeAssert
		}

		for i := chunk.Begin(); i < chunk.End(); i++ {

			// Eq 16.1: Invert the round message private key
			inverse(s.Grp, s.MessagePrecomputation.Get(i), s.MessagePrecomputation.Get(i))

			// Eq 16.2: Invert the round associated data private key
			inverse(s.Grp, s.ADPrecomputation.Get(i), s.ADPrecomputation.Get(i))

		}
		return nil
	},
	Cryptop:    cryptops.Inverse,
	NumThreads: 5,
	InputSize:  services.AUTO_INPUTSIZE,
	Name:       "StripInverse",
}

var StripMul2 = services.Module{
	// Runs mul2 for cypher message and cypher associated data
	Adapt: func(streamInput services.Stream, cryptop cryptops.Cryptop, chunk services.Chunk) error {
		s, ok := streamInput.(*StripStream)
		mul2, ok2 := cryptop.(cryptops.Mul2Prototype)

		if !ok || !ok2 {
			return services.InvalidTypeAssert
		}

		for i := chunk.Begin(); i < chunk.End(); i++ {

			// Eq 16.1: Use the inverted round message private key to remove the
			//          homomorphic encryption from encrypted message key and reveal
			//          the message precomputation
			mul2(s.Grp, s.CypherMsg.Get(i), s.MessagePrecomputation.Get(i))

			// Eq 16.2: Use the inverted round associated data private key to remove
			//          the homomorphic encryption from encrypted associated data key
			//          and reveal the associated data precomputation
			mul2(s.Grp, s.CypherAD.Get(i), s.ADPrecomputation.Get(i))

		}
		return nil
	},
	Cryptop:    cryptops.Mul2,
	NumThreads: 5,
	InputSize:  services.AUTO_INPUTSIZE,
	Name:       "StripMul2",
}

// Called to initialize the graph. Conforms to graphs.Initialize function type
func InitStripGraph(gc services.GraphGenerator) *services.Graph {
	graph := gc.NewGraph("PrecompStrip", &StripStream{})

	stripInverse := StripInverse.DeepCopy()
	stripMul2 := StripMul2.DeepCopy()

	graph.First(stripInverse)
	graph.Connect(stripInverse, stripMul2)
	graph.Last(stripMul2)

	return graph
}
