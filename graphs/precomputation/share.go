////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package precomputation

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/node"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/services"
)

// This file implements the Graph for the Precomputation Share phase
// Share creates executes the group diffie Hellman that acts as the bedrock of
// CMIX's security.  This implementation is not secure.  It is meant to be used with a batch size of 1

// ShareStream holds data containing keys and inputs used by share
type ShareStream struct {
	Grp                    *cyclic.Group
	PartialPublicCypherKey *cyclic.Int
	Z                      *cyclic.Int
}

// GetName returns stream name
func (s *ShareStream) GetName() string {
	return "PrecompShareStream"
}

// Link binds stream to state objects in round
func (s *ShareStream) Link(grp *cyclic.Group, batchSize uint32, source ...interface{}) {
	roundBuffer := source[0].(*round.Buffer)

	s.Grp = grp
	s.Z = roundBuffer.Z

	s.PartialPublicCypherKey = s.Grp.NewInt(1)
}

// Input initializes stream inputs from slot
func (s *ShareStream) Input(index uint32, slot *mixmessages.Slot) error {

	if !s.Grp.BytesInside(slot.PartialRoundPublicCypherKey) {
		return node.ErrOutsideOfGroup
	}

	s.Grp.SetBytes(s.PartialPublicCypherKey, slot.PartialRoundPublicCypherKey)
	return nil
}

// Output returns a cmix slot message
func (s *ShareStream) Output(index uint32) *mixmessages.Slot {

	return &mixmessages.Slot{
		PartialRoundPublicCypherKey: s.PartialPublicCypherKey.Bytes(),
	}
}

// ShareExp is sole module in Precomputation Decrypt implementing cryptops.Elgamal
var ShareExp = services.Module{
	// Multiplies in own Encrypted Keys and Partial Cypher Texts
	Adapt: func(streamInput services.Stream, cryptop cryptops.Cryptop, chunk services.Chunk) error {
		s, ok := streamInput.(*ShareStream)
		exp, ok2 := cryptop.(cryptops.ExpPrototype)

		if !ok || !ok2 {
			return services.InvalidTypeAssert
		}

		for i := chunk.Begin(); i < chunk.End(); i++ {
			// Exponentiates the partial round cypher key by the node's round private key (Z)
			exp(s.Grp, s.PartialPublicCypherKey, s.Z, s.PartialPublicCypherKey)
		}
		return nil
	},
	Cryptop:    cryptops.Exp,
	NumThreads: 1,
	InputSize:  1,
	Name:       "Share",
}

// InitShareGraph is called to initialize the graph. Conforms to graphs.Initialize function type
func InitShareGraph(gc services.GraphGenerator) *services.Graph {
	//Share is special and  must have an input size of 1.  The graph generator must allow for that.
	if gc.GetMinInputSize() != 1 {
		jww.FATAL.Panicf("Share must have an input size of one, " +
			"cannot generate off generator which requires larger")
	}

	g := gc.NewGraph("Share", &ShareStream{})

	shareExp := ShareExp.DeepCopy()

	g.First(shareExp)
	g.Last(shareExp)

	return g
}
