////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package precomputation

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/internals/round"
	"gitlab.com/elixxir/server/services"
)

// This file implements the Graph for the Precomputation Strip phase.
// Strip phase inverts the Round Private Keys and removes the
// homomorphic encryption from the encrypted keys, revealing completed
// precomputation

// StripStream holds data containing private key from encrypt and
// inputs used by strip
type StripStream struct {
	Grp *cyclic.Group

	// Link to round object
	PayloadAPrecomputation          *cyclic.IntBuffer
	PayloadBPrecomputation          *cyclic.IntBuffer
	EncryptedPayloadAPrecomputation []*cyclic.Int
	EncryptedPayloadBPrecomputation []*cyclic.Int

	// Unique to stream
	CypherPayloadA *cyclic.IntBuffer
	CypherPayloadB *cyclic.IntBuffer

	RevealStream
}

// GetName returns stream name
func (ss *StripStream) GetName() string {
	return "PrecompStripStream"
}

// Link binds stream to state objects in round
func (ss *StripStream) Link(grp *cyclic.Group, batchSize uint32,
	source ...interface{}) {
	roundBuffer := source[0].(*round.Buffer)

	ss.LinkPrecompStripStream(grp, batchSize, roundBuffer,
		grp.NewIntBuffer(batchSize, grp.NewInt(1)),
		grp.NewIntBuffer(batchSize, grp.NewInt(1)))

}

func (ss *StripStream) LinkPrecompStripStream(grp *cyclic.Group,
	batchSize uint32, roundBuf *round.Buffer,
	cypherPayloadA, keysPayloadB *cyclic.IntBuffer) {

	ss.Grp = grp

	ss.PayloadAPrecomputation = roundBuf.PayloadAPrecomputation.GetSubBuffer(
		0, batchSize)
	ss.PayloadBPrecomputation = roundBuf.PayloadBPrecomputation.GetSubBuffer(
		0, batchSize)

	ss.EncryptedPayloadAPrecomputation = roundBuf.PermutedPayloadAKeys
	ss.EncryptedPayloadBPrecomputation = roundBuf.PermutedPayloadBKeys

	ss.CypherPayloadA = cypherPayloadA
	ss.CypherPayloadB = keysPayloadB

	ss.RevealStream.LinkStream(grp, batchSize, roundBuf, ss.CypherPayloadA,
		ss.CypherPayloadB)
}

type stripSubstreamInterface interface {
	GetStripSubStream() *StripStream
}

// getSubStream implements reveal interface to return stream object
func (ss *StripStream) GetStripSubStream() *StripStream {
	return ss
}

// Input initializes stream inputs from slot
func (ss *StripStream) Input(index uint32, slot *mixmessages.Slot) error {

	if index >= uint32(ss.CypherPayloadA.Len()) {
		return services.ErrOutsideOfBatch
	}

	if !ss.Grp.BytesInside(slot.PartialPayloadACypherText,
		slot.PartialPayloadBCypherText) {
		return services.ErrOutsideOfGroup
	}

	ss.Grp.SetBytes(ss.CypherPayloadA.Get(index), slot.PartialPayloadACypherText)
	ss.Grp.SetBytes(ss.CypherPayloadB.Get(index),
		slot.PartialPayloadBCypherText)

	return nil
}

// Output returns a cmix slot message
func (ss *StripStream) Output(index uint32) *mixmessages.Slot {
	return &mixmessages.Slot{
		Index: index,
		EncryptedPayloadAKeys: ss.PayloadAPrecomputation.Get(
			index).Bytes(),
		EncryptedPayloadBKeys: ss.PayloadBPrecomputation.Get(index).Bytes(),
	}
}

// StripInverse is a module in precomputation strip implementing
// cryptops.Inverse
var StripInverse = services.Module{
	// Runs root coprime for cypher texts
	Adapt: func(streamInput services.Stream, cryptop cryptops.Cryptop,
		chunk services.Chunk) error {
		sssi, ok := streamInput.(stripSubstreamInterface)
		inverse, ok2 := cryptop.(cryptops.InversePrototype)

		if !ok || !ok2 {
			return services.InvalidTypeAssert
		}

		ss := sssi.GetStripSubStream()

		for i := chunk.Begin(); i < chunk.End(); i++ {

			// Eq 16.1: Invert the round payload A private key
			inverse(ss.Grp, ss.EncryptedPayloadAPrecomputation[i], ss.PayloadAPrecomputation.Get(i))

			// Eq 16.2: Invert the round payload B private key
			inverse(ss.Grp, ss.EncryptedPayloadBPrecomputation[i], ss.PayloadBPrecomputation.Get(i))

		}
		return nil
	},
	Cryptop:    cryptops.Inverse,
	NumThreads: 5,
	InputSize:  services.AutoInputSize,
	Name:       "StripInverse",
}

// StripMul2 is a module in precomputation strip implementing cryptops.mul2
var StripMul2 = services.Module{
	// Runs mul2 for cypher texts
	Adapt: func(streamInput services.Stream, cryptop cryptops.Cryptop,
		chunk services.Chunk) error {
		sssi, ok := streamInput.(stripSubstreamInterface)
		mul2, ok2 := cryptop.(cryptops.Mul2Prototype)

		if !ok || !ok2 {
			return services.InvalidTypeAssert
		}

		ss := sssi.GetStripSubStream()

		for i := chunk.Begin(); i < chunk.End(); i++ {
			// Eq 16.1: Use the first payload's inverted round private key
			//          to remove the homomorphic encryption from
			//          first payload's encrypted key and reveal the
			//          first payload's precomputation

			mul2(ss.Grp, ss.CypherPayloadA.Get(i), ss.PayloadAPrecomputation.Get(i))

			// Eq 16.2: Use the second payload's inverted round
			//          private key to remove the homomorphic
			//          encryption from the second payload's encrypted
			//          key and reveal the second payload's precomputation
			mul2(ss.Grp, ss.CypherPayloadB.Get(i), ss.PayloadBPrecomputation.Get(i))
		}
		return nil
	},
	Cryptop:    cryptops.Mul2,
	NumThreads: services.AutoNumThreads,
	InputSize:  services.AutoInputSize,
	Name:       "StripMul2",
}

// InitStripGraph to initialize the graph. Conforms to graphs.Initialize function type
func InitStripGraph(gc services.GraphGenerator) *services.Graph {
	graph := gc.NewGraph("PrecompStrip", &StripStream{})

	reveal := RevealRootCoprime.DeepCopy()
	stripInverse := StripInverse.DeepCopy()
	stripMul2 := StripMul2.DeepCopy()

	graph.First(reveal)
	graph.Connect(reveal, stripInverse)
	graph.Connect(stripInverse, stripMul2)
	graph.Last(stripMul2)

	return graph
}
