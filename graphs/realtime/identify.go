////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package realtime

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/services"
)

type IdentifyStream struct {
	Grp *cyclic.Group

	EcrPayloadA *cyclic.IntBuffer
	EcrPayloadB *cyclic.IntBuffer

	// inputs to the phase
	EcrPayloadAPermuted []*cyclic.Int
	EcrPayloadBPermuted []*cyclic.Int

	PayloadAPrecomputation *cyclic.IntBuffer
	PayloadBPrecomputation *cyclic.IntBuffer

	PermuteStream
}

// GetName returns the name of the stream for debugging purposes.
func (is *IdentifyStream) GetName() string {
	return "RealtimeIdentifyStream"
}

// Link binds stream data to state objects in round.
func (is *IdentifyStream) Link(grp *cyclic.Group, batchSize uint32, source ...interface{}) {
	roundBuffer := source[0].(*round.Buffer)

	is.LinkIdentifyStreams(grp, batchSize, roundBuffer,
		grp.NewIntBuffer(batchSize, grp.NewInt(1)),
		grp.NewIntBuffer(batchSize, grp.NewInt(1)),
		make([]*cyclic.Int, batchSize),
		make([]*cyclic.Int, batchSize))
}

// LinkRealtimePermuteStreams binds stream data.
func (is *IdentifyStream) LinkIdentifyStreams(grp *cyclic.Group, batchSize uint32, round *round.Buffer,
	ecrPayloadA, ecrPayloadB *cyclic.IntBuffer, permPayloadA, permPayloadB []*cyclic.Int) {

	is.Grp = grp

	is.EcrPayloadA = ecrPayloadA
	is.EcrPayloadB = ecrPayloadB

	is.PayloadAPrecomputation = round.PayloadAPrecomputation.GetSubBuffer(0, batchSize)
	is.PayloadBPrecomputation = round.PayloadBPrecomputation.GetSubBuffer(0, batchSize)

	is.EcrPayloadAPermuted = permPayloadA
	is.EcrPayloadBPermuted = permPayloadB

	is.LinkRealtimePermuteStreams(grp, batchSize, round,
		is.EcrPayloadA,
		is.EcrPayloadB,
		is.EcrPayloadAPermuted,
		is.EcrPayloadBPermuted)

}

type identifyStreamInterface interface {
	getIdentifyStream() *IdentifyStream
}

func (is *IdentifyStream) getIdentifyStream() *IdentifyStream {
	return is
}

// Input initializes stream inputs from slot.
func (is *IdentifyStream) Input(index uint32, slot *mixmessages.Slot) error {
	if index >= uint32(is.EcrPayloadA.Len()) {
		return services.ErrOutsideOfBatch
	}

	if !is.Grp.BytesInside(slot.PayloadA, slot.PayloadB) {
		return services.ErrOutsideOfGroup
	}

	is.Grp.SetBytes(is.EcrPayloadA.Get(index), slot.PayloadA)
	is.Grp.SetBytes(is.EcrPayloadB.Get(index), slot.PayloadB)

	return nil
}

// Output returns a message with the stream data.
func (is *IdentifyStream) Output(index uint32) *mixmessages.Slot {
	byteLen := uint64(len(is.Grp.GetPBytes()))
	return &mixmessages.Slot{
		Index:          index,
		PayloadA: is.EcrPayloadAPermuted[index].LeftpadBytes(byteLen),
		PayloadB: is.EcrPayloadBPermuted[index].LeftpadBytes(byteLen),
	}
}

// Module implementing cryptops.Mul2.
var IdentifyMul2 = services.Module{
	// Multiplies in own Encrypted Keys and Partial Cypher Texts
	Adapt: func(stream services.Stream, cryptop cryptops.Cryptop, chunk services.Chunk) error {
		isi, ok := stream.(identifyStreamInterface)
		mul2, ok2 := cryptop.(cryptops.Mul2Prototype)

		if !ok || !ok2 {
			return services.InvalidTypeAssert
		}

		is := isi.getIdentifyStream()

		for i := chunk.Begin(); i < chunk.End(); i++ {
			// Multiply the encrypted message by the precomputation to decrypt it
			mul2(is.Grp, is.PayloadAPrecomputation.Get(i), is.EcrPayloadAPermuted[i])
			// Multiply the encrypted associated data by the precomputation to decrypt it
			mul2(is.Grp, is.PayloadBPrecomputation.Get(i), is.EcrPayloadBPermuted[i])
		}
		return nil
	},
	Cryptop:        cryptops.Mul2,
	NumThreads:     services.AutoNumThreads,
	InputSize:      services.AutoInputSize,
	StartThreshold: 1.0,
	Name:           "Identify",
}

// InitIdentifyGraph initializes and returns a new graph.
func InitIdentifyGraph(gc services.GraphGenerator) *services.Graph {
	g := gc.NewGraph("RealtimeIdentify", &IdentifyStream{})

	permuteMul2 := PermuteMul2.DeepCopy()
	identifyMul2 := IdentifyMul2.DeepCopy()

	g.First(permuteMul2)
	g.Connect(permuteMul2, identifyMul2)
	g.Last(identifyMul2)

	return g
}
