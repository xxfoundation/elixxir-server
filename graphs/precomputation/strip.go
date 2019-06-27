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
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/services"
)

// This file implements the Graph for the Precomputation Strip phase.
// Strip phase inverts the Round Private Keys and removes the
// homomorphic encryption from the encrypted message keys and
// encrypted associated data keys, revealing completed precomputation

// StripStream holds data containing private key from encrypt and
// inputs used by strip
type StripStream struct {
	Grp *cyclic.Group

	// Link to round object
	MessagePrecomputation          *cyclic.IntBuffer
	ADPrecomputation               *cyclic.IntBuffer
	EncryptedMessagePrecomputation []*cyclic.Int
	EncryptedADPrecomputation      []*cyclic.Int

	// Unique to stream
	CypherMsg *cyclic.IntBuffer
	CypherAD  *cyclic.IntBuffer

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
	cypherMsg, cypherAD *cyclic.IntBuffer) {

	ss.Grp = grp

	ss.MessagePrecomputation = roundBuf.MessagePrecomputation.GetSubBuffer(
		0, batchSize)
	ss.ADPrecomputation = roundBuf.ADPrecomputation.GetSubBuffer(
		0, batchSize)
	ss.EncryptedMessagePrecomputation = roundBuf.PermutedMessageKeys
	ss.EncryptedADPrecomputation = roundBuf.PermutedADKeys

	ss.CypherMsg = cypherMsg
	ss.CypherAD = cypherAD

	ss.RevealStream.LinkStream(grp, batchSize, roundBuf, ss.CypherMsg,
		ss.CypherAD)
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

	if index >= uint32(ss.CypherMsg.Len()) {
		return services.ErrOutsideOfBatch
	}

	if !ss.Grp.BytesInside(slot.PartialMessageCypherText,
		slot.PartialAssociatedDataCypherText) {
		return services.ErrOutsideOfGroup
	}

	ss.Grp.SetBytes(ss.CypherMsg.Get(index), slot.PartialMessageCypherText)
	ss.Grp.SetBytes(ss.CypherAD.Get(index),
		slot.PartialAssociatedDataCypherText)

	return nil
}

// Output returns a cmix slot message
func (ss *StripStream) Output(index uint32) *mixmessages.Slot {
	return &mixmessages.Slot{
		Index: index,
		PartialMessageCypherText: ss.MessagePrecomputation.Get(
			index).Bytes(),
		PartialAssociatedDataCypherText: ss.ADPrecomputation.Get(index).Bytes(),
	}
}

// StripInverse is a module in precomputation strip implementing
// cryptops.Inverse
var StripInverse = services.Module{
	// Runs root coprime for cypher message and cypher associated data
	Adapt: func(streamInput services.Stream, cryptop cryptops.Cryptop,
		chunk services.Chunk) error {
		sssi, ok := streamInput.(stripSubstreamInterface)
		inverse, ok2 := cryptop.(cryptops.InversePrototype)

		if !ok || !ok2 {
			return services.InvalidTypeAssert
		}

		ss := sssi.GetStripSubStream()

		for i := chunk.Begin(); i < chunk.End(); i++ {

			// Eq 16.1: Invert the round message private key
			inverse(ss.Grp, ss.EncryptedMessagePrecomputation[i], ss.MessagePrecomputation.Get(i))

			// Eq 16.2: Invert the round associated data private key
			inverse(ss.Grp, ss.EncryptedADPrecomputation[i], ss.ADPrecomputation.Get(i))

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
	// Runs mul2 for cypher message and cypher associated data
	Adapt: func(streamInput services.Stream, cryptop cryptops.Cryptop,
		chunk services.Chunk) error {
		sssi, ok := streamInput.(stripSubstreamInterface)
		mul2, ok2 := cryptop.(cryptops.Mul2Prototype)

		if !ok || !ok2 {
			return services.InvalidTypeAssert
		}

		ss := sssi.GetStripSubStream()

		for i := chunk.Begin(); i < chunk.End(); i++ {
			// Eq 16.1: Use the inverted round message private key
			//          to remove the homomorphic encryption from
			//          encrypted message key and reveal the message
			//          precomputation

			mul2(ss.Grp, ss.CypherMsg.Get(i), ss.MessagePrecomputation.Get(i))

			// Eq 16.2: Use the inverted round associated data
			//          private key to remove the homomorphic
			//          encryption from encrypted associated data
			//          key and reveal the associated data
			//          precomputation
			mul2(ss.Grp, ss.CypherAD.Get(i), ss.ADPrecomputation.Get(i))
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
