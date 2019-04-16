package precomputation

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/node"
	"gitlab.com/elixxir/server/services"
)

// This file implements the Graph for the Precomputation Reveal phase.
// The reveal phase removes cypher keys from the message and
// associated data cypher text, revealing the private keys for the round.

// Stream holding data containing private key from encrypt and inputs used by strip
type RevealStream struct {
	Grp             *cyclic.Group
	CypherPublicKey *cyclic.Int

	// Link to round object
	Z *cyclic.Int

	// Unique to stream
	CypherMsg *cyclic.IntBuffer
	CypherAD  *cyclic.IntBuffer
}

func (s *RevealStream) GetName() string {
	return "PrecompRevealStream"
}

func (s *RevealStream) Link(batchSize uint32, source interface{}) {
	round := source.(*node.RoundBuffer)

	s.Grp = round.Grp
	s.CypherPublicKey = round.CypherPublicKey

	s.Z = round.Z

	s.CypherMsg = s.Grp.NewIntBuffer(batchSize, s.Grp.NewInt(1))
	s.CypherAD = s.Grp.NewIntBuffer(batchSize, s.Grp.NewInt(1))
}

func (s *RevealStream) Input(index uint32, slot *mixmessages.CmixSlot) error {

	if !s.Grp.BytesInside(slot.PartialMessageCypherText, slot.PartialAssociatedDataCypherText) {
		return node.ErrOutsideOfGroup
	}

	s.Grp.SetBytes(s.CypherMsg.Get(index), slot.PartialMessageCypherText)
	s.Grp.SetBytes(s.CypherAD.Get(index), slot.PartialAssociatedDataCypherText)
	return nil
}

func (s *RevealStream) Output(index uint32) *mixmessages.CmixSlot {

	return &mixmessages.CmixSlot{
		PartialMessageCypherText:        s.CypherMsg.Get(index).Bytes(),
		PartialAssociatedDataCypherText: s.CypherAD.Get(index).Bytes(),
	}
}

// Module in precomputation reveeal implementing cryptops.RootCoprimePrototype
var RevealRootCoprime = services.Module{
	// Runs root coprime for cypher message and cypher associated data
	Adapt: func(streamInput services.Stream, cryptop cryptops.Cryptop, chunk services.Chunk) error {
		s, ok := streamInput.(*RevealStream)
		rootCoprime, ok2 := cryptop.(cryptops.RootCoprimePrototype)

		if !ok || !ok2 {
			return services.InvalidTypeAssert
		}

		for i := chunk.Begin(); i < chunk.End(); i++ {
			// Execute rootCoprime on the keys for the Message
			// Eq 15.11 Root by cypher key to remove one layer of homomorphic
			// encryption from partially encrypted message cypher text.
			rootCoprime(s.Grp, s.CypherMsg.Get(i), s.Z, s.CypherMsg.Get(i))

			// Execute rootCoprime on the keys for the associated data
			// Eq 15.13 Root by cypher key to remove one layer of homomorphic
			// encryption from partially encrypted associated data cypher text.
			rootCoprime(s.Grp, s.CypherAD.Get(i), s.Z, s.CypherAD.Get(i))

		}
		return nil
	},
	Cryptop:    cryptops.RootCoprime,
	NumThreads: 5,
	InputSize:  services.AUTO_INPUTSIZE,
	Name:       "RevealRootCoprime",
}

// Called to initialize the graph. Conforms to graphs.Initialize function type
func InitRevealGraph(errorHandler services.ErrorCallback) *services.Graph {
	graph := services.NewGraph("PrecompReveal", errorHandler, &RevealStream{})

	revealRootCoprime := RevealRootCoprime.DeepCopy()

	graph.First(revealRootCoprime)
	graph.Last(revealRootCoprime)

	return graph
}
