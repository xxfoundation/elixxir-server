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
func (ss *ShareStream) GetName() string {
	return "PrecompShareStream"
}

// Link binds stream to state objects in round
func (ss *ShareStream) Link(grp *cyclic.Group, batchSize uint32, source ...interface{}) {
	roundBuffer := source[0].(*round.Buffer)

	ss.LinkShareStream(grp, batchSize, roundBuffer)
}

// Link binds stream to state objects in round
func (ss *ShareStream) LinkShareStream(grp *cyclic.Group, batchSize uint32, roundBuffer *round.Buffer) {
	ss.Grp = grp
	ss.Z = roundBuffer.Z

	ss.PartialPublicCypherKey = ss.Grp.NewInt(1)
}

// getSubStream implements reveal interface to return stream object
func (ss *ShareStream) GetSubStream() *ShareStream {
	return ss
}

// Input initializes stream inputs from slot
func (ss *ShareStream) Input(index uint32, slot *mixmessages.Slot) error {

	if !ss.Grp.BytesInside(slot.PartialRoundPublicCypherKey) {
		return services.ErrOutsideOfGroup
	}

	ss.Grp.SetBytes(ss.PartialPublicCypherKey, slot.PartialRoundPublicCypherKey)
	return nil
}

// Output returns a cmix slot message
func (ss *ShareStream) Output(index uint32) *mixmessages.Slot {
	jww.ERROR.Printf("in share streams")
	return &mixmessages.Slot{
		Index:                       index,
		PartialRoundPublicCypherKey: ss.PartialPublicCypherKey.Bytes(),
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

	g.OverrideBatchSize(1)

	return g
}
