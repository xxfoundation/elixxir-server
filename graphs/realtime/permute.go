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
	"gitlab.com/elixxir/server/graphs"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/services"
)

type PermuteStream struct {
	Grp *cyclic.Group

	S *cyclic.IntBuffer
	V *cyclic.IntBuffer

	EcrPayloadA *cyclic.IntBuffer
	EcrPayloadB *cyclic.IntBuffer

	PayloadAPermuted []*cyclic.Int
	PayloadBPermuted []*cyclic.Int
}

// GetName returns the name of the stream for debugging purposes.
func (ps *PermuteStream) GetName() string {
	return "RealtimePermuteStream"
}

// Link binds stream data to state objects in round.
func (ps *PermuteStream) Link(grp *cyclic.Group, batchSize uint32, source ...interface{}) {
	roundBuffer := source[0].(*round.Buffer)

	ps.LinkRealtimePermuteStreams(grp, batchSize, roundBuffer,
		grp.NewIntBuffer(batchSize, grp.NewInt(1)),
		grp.NewIntBuffer(batchSize, grp.NewInt(1)),
		make([]*cyclic.Int, batchSize),
		make([]*cyclic.Int, batchSize))
}

// LinkPermuteStreams binds stream data.
func (ps *PermuteStream) LinkRealtimePermuteStreams(grp *cyclic.Group,
	batchSize uint32, roundBuffer *round.Buffer, msg, ad *cyclic.IntBuffer, msgPerm,
	adPerm []*cyclic.Int) {
	ps.Grp = grp

	ps.S = roundBuffer.S.GetSubBuffer(0, batchSize)
	ps.V = roundBuffer.V.GetSubBuffer(0, batchSize)

	ps.EcrPayloadA = msg
	ps.EcrPayloadB = ad

	ps.PayloadAPermuted = msgPerm
	ps.PayloadBPermuted = adPerm

	graphs.PrecanPermute(roundBuffer.Permutations,
		graphs.PermuteIO{Input: ps.EcrPayloadA, Output: ps.PayloadAPermuted},
		graphs.PermuteIO{Input: ps.EcrPayloadB, Output: ps.PayloadBPermuted})

}

// PermuteStream conforms to this interface.
type permuteSubStreamInterface interface {
	getPermuteSubStream() *PermuteStream
}

// getPermuteSubStream returns the sub-stream, used to return an embedded struct
// off an interface.
func (ps *PermuteStream) getPermuteSubStream() *PermuteStream {
	return ps
}

// Input initializes stream inputs from slot.
func (ps *PermuteStream) Input(index uint32, slot *mixmessages.Slot) error {
	if index >= uint32(ps.EcrPayloadA.Len()) {
		return services.ErrOutsideOfBatch
	}

	if !ps.Grp.BytesInside(slot.PayloadA, slot.PayloadB) {
		return services.ErrOutsideOfGroup
	}

	ps.Grp.SetBytes(ps.EcrPayloadA.Get(index), slot.PayloadA)
	ps.Grp.SetBytes(ps.EcrPayloadB.Get(index), slot.PayloadB)

	return nil
}

// Output returns a message with the stream data.
func (ps *PermuteStream) Output(index uint32) *mixmessages.Slot {
	return &mixmessages.Slot{
		Index:    index,
		PayloadA: ps.PayloadAPermuted[index].Bytes(),
		PayloadB: ps.PayloadBPermuted[index].Bytes(),
	}
}

// Module implementing cryptops.Mul2.
var PermuteMul2 = services.Module{
	Adapt: func(stream services.Stream, cryptop cryptops.Cryptop,
		chunk services.Chunk) error {
		psi, ok1 := stream.(permuteSubStreamInterface)
		mul2, ok2 := cryptop.(cryptops.Mul2Prototype)

		if !ok1 || !ok2 {
			return services.InvalidTypeAssert
		}

		ps := psi.getPermuteSubStream()

		for i := chunk.Begin(); i < chunk.End(); i++ {
			mul2(ps.Grp, ps.S.Get(i), ps.EcrPayloadA.Get(i))

			mul2(ps.Grp, ps.V.Get(i), ps.EcrPayloadB.Get(i))
		}

		return nil
	},
	Cryptop:    cryptops.Mul2,
	InputSize:  services.AutoInputSize,
	Name:       "PermuteRealtime",
	NumThreads: services.AutoNumThreads,
}

// InitPermuteGraph initializes and returns a new graph.
func InitPermuteGraph(gc services.GraphGenerator) *services.Graph {
	gcPermute := graphs.ModifyGraphGeneratorForPermute(gc)
	g := gcPermute.NewGraph("RealtimePermute", &PermuteStream{})

	mul2 := PermuteMul2.DeepCopy()

	g.First(mul2)
	g.Last(mul2)

	return g
}
