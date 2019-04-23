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

func (ps *PermuteStream) GetName() string {
	return "RealtimePermuteStream"
}

func (ps *PermuteStream) Link(batchSize uint32, source interface{}) {
	round := source.(*node.RoundBuffer)

	ps.Grp = round.Grp

	ps.S = round.S.GetSubBuffer(0, batchSize)
	ps.V = round.V.GetSubBuffer(0, batchSize)

	ps.Msg = ps.Grp.NewIntBuffer(batchSize, ps.Grp.NewInt(1))
	ps.AD = ps.Grp.NewIntBuffer(batchSize, ps.Grp.NewInt(1))

	ps.MsgPermuted = make([]*cyclic.Int, batchSize)
	ps.ADPermuted = make([]*cyclic.Int, batchSize)

	ps.PermuteSubStream.LinkStreams(batchSize, round.Permutations,
		graphs.PermuteIO{Input: ps.Msg, Output: ps.MsgPermuted},
		graphs.PermuteIO{Input: ps.AD, Output: ps.ADPermuted})
}

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

func (ps *PermuteStream) Output(index uint32) *mixmessages.CmixSlot {
	return &mixmessages.CmixSlot{
		MessagePayload: ps.MsgPermuted[index].Bytes(),
		AssociatedData: ps.ADPermuted[index].Bytes(),
	}
}

var PermuteMul2 = services.Module{
	Adapt: func(stream services.Stream, cryptop cryptops.Cryptop, chunk services.Chunk) error {
		ps, ok1 := stream.(*PermuteStream)
		mul2, ok2 := cryptop.(cryptops.Mul2Prototype)

		if !ok1 || !ok2 {
			return services.InvalidTypeAssert
		}

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

func InitPermuteGraph(gc services.GraphGenerator) *services.Graph {
	g := gc.NewGraph("RealtimePermute", &PermuteStream{})

	mul2 := PermuteMul2.DeepCopy()
	permute := graphs.Permute.DeepCopy()

	g.First(mul2)
	g.Connect(mul2, permute)
	g.Last(permute)

	return g
}
