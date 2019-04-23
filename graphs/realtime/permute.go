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
	"gitlab.com/elixxir/server/node"
	"gitlab.com/elixxir/server/services"
)

type PermuteStream struct {
	Grp *cyclic.Group

	S *cyclic.IntBuffer
	V *cyclic.IntBuffer

	Msg *cyclic.IntBuffer
	AD  *cyclic.IntBuffer

	MsgPermuted []*cyclic.Int
	ADPermuted  []*cyclic.Int

	graphs.PermuteSubStream
}

// GetName returns the name of the stream for debugging purposes.
func (ps *PermuteStream) GetName() string {
	return "RealtimePermuteStream"
}

// Link binds stream data to state objects in round.
func (ps *PermuteStream) Link(batchSize uint32, source interface{}) {
	round := source.(*node.RoundBuffer)

	ps.LinkPermuteStreams(batchSize, round,
		round.Grp.NewIntBuffer(batchSize, round.Grp.NewInt(1)),
		round.Grp.NewIntBuffer(batchSize, round.Grp.NewInt(1)),
		make([]*cyclic.Int, batchSize),
		make([]*cyclic.Int, batchSize))
}

// LinkPermuteStreams binds stream data.
func (ps *PermuteStream) LinkPermuteStreams(batchSize uint32,
	round *node.RoundBuffer, msg, ad *cyclic.IntBuffer, msgPerm,
	adPerm []*cyclic.Int) {
	ps.Grp = round.Grp

	ps.S = round.S.GetSubBuffer(0, batchSize)
	ps.V = round.V.GetSubBuffer(0, batchSize)

	ps.Msg = msg
	ps.AD = ad

	ps.MsgPermuted = msgPerm
	ps.ADPermuted = adPerm

	ps.PermuteSubStream.LinkStreams(batchSize, round.Permutations,
		graphs.PermuteIO{Input: ps.Msg, Output: ps.MsgPermuted},
		graphs.PermuteIO{Input: ps.AD, Output: ps.ADPermuted})

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
func (ps *PermuteStream) Input(index uint32, slot *mixmessages.CmixSlot) error {
	if index >= uint32(ps.Msg.Len()) {
		return node.ErrOutsideOfBatch
	}

	if !ps.Grp.BytesInside(slot.MessagePayload, slot.AssociatedData) {
		return node.ErrOutsideOfGroup
	}

	ps.Grp.SetBytes(ps.Msg.Get(index), slot.MessagePayload)
	ps.Grp.SetBytes(ps.AD.Get(index), slot.AssociatedData)

	return nil
}

// Output returns a message with the stream data.
func (ps *PermuteStream) Output(index uint32) *mixmessages.CmixSlot {
	return &mixmessages.CmixSlot{
		MessagePayload: ps.MsgPermuted[index].Bytes(),
		AssociatedData: ps.ADPermuted[index].Bytes(),
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
			mul2(ps.Grp, ps.S.Get(i), ps.Msg.Get(i))

			mul2(ps.Grp, ps.V.Get(i), ps.AD.Get(i))
		}

		return nil
	},
	Cryptop:    cryptops.Mul2,
	InputSize:  services.AUTO_INPUTSIZE,
	Name:       "PermuteRealtime",
	NumThreads: 5,
}

// InitPermuteGraph initializes and returns a new graph.
func InitPermuteGraph(gc services.GraphGenerator) *services.Graph {
	g := gc.NewGraph("RealtimePermute", &PermuteStream{})

	mul2 := PermuteMul2.DeepCopy()
	permute := graphs.Permute.DeepCopy()

	g.First(mul2)
	g.Connect(mul2, permute)
	g.Last(permute)

	return g
}
