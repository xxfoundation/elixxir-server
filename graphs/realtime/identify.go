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

type IdentifyStream struct {
	Grp *cyclic.Group

	EcrMsg *cyclic.IntBuffer
	EcrAD  *cyclic.IntBuffer

	// inputs to the phase
	EcrMsgPermuted []*cyclic.Int
	EcrADPermuted  []*cyclic.Int

	MsgPrecomputation *cyclic.IntBuffer
	ADPrecomputation  *cyclic.IntBuffer

	PermuteStream
}

// GetName returns the name of the stream for debugging purposes.
func (is *IdentifyStream) GetName() string {
	return "RealtimeIdentifyStream"
}

// Link binds stream data to state objects in round.
func (is *IdentifyStream) Link(batchSize uint32, source interface{}) {
	round := source.(*node.RoundBuffer)

	is.LinkIdentifyStreams(batchSize, round,
		round.Grp.NewIntBuffer(batchSize, round.Grp.NewInt(1)),
		round.Grp.NewIntBuffer(batchSize, round.Grp.NewInt(1)),
		make([]*cyclic.Int, batchSize),
		make([]*cyclic.Int, batchSize))
}

// LinkRealtimePermuteStreams binds stream data.
func (is *IdentifyStream) LinkIdentifyStreams(batchSize uint32, round *node.RoundBuffer,
	ecrMsg, ecrAD *cyclic.IntBuffer, permMsg, permAD []*cyclic.Int) {

	is.Grp = round.Grp

	is.EcrMsg = ecrMsg
	is.EcrAD = ecrAD

	is.MsgPrecomputation = round.MessagePrecomputation.GetSubBuffer(0, batchSize)
	is.ADPrecomputation = round.ADPrecomputation.GetSubBuffer(0, batchSize)

	is.EcrMsgPermuted = permMsg
	is.EcrADPermuted = permAD

	is.LinkRealtimePermuteStreams(batchSize, round,
		is.EcrMsg,
		is.EcrAD,
		is.EcrMsgPermuted,
		is.EcrADPermuted)

}

type identifyStreamInterface interface {
	getIdentifyStream() *IdentifyStream
}

func (is *IdentifyStream) getIdentifyStream() *IdentifyStream {
	return is
}

// Input initializes stream inputs from slot.
func (is *IdentifyStream) Input(index uint32, slot *mixmessages.CmixSlot) error {
	if index >= uint32(is.EcrMsg.Len()) {
		return node.ErrOutsideOfBatch
	}

	if !is.Grp.BytesInside(slot.MessagePayload, slot.AssociatedData) {
		return node.ErrOutsideOfGroup
	}

	is.Grp.SetBytes(is.EcrMsg.Get(index), slot.MessagePayload)
	is.Grp.SetBytes(is.EcrAD.Get(index), slot.AssociatedData)

	return nil
}

// Output returns a message with the stream data.
func (is *IdentifyStream) Output(index uint32) *mixmessages.CmixSlot {
	return &mixmessages.CmixSlot{
		MessagePayload: is.EcrMsgPermuted[index].Bytes(),
		AssociatedData: is.EcrADPermuted[index].Bytes(),
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
			mul2(is.Grp, is.MsgPrecomputation.Get(i), is.EcrMsgPermuted[i])
			// Multiply the encrypted associated data by the precomputation to decrypt it
			mul2(is.Grp, is.ADPrecomputation.Get(i), is.EcrADPermuted[i])
		}
		return nil
	},
	Cryptop:        cryptops.Mul2,
	NumThreads:     5,
	InputSize:      services.AUTO_INPUTSIZE,
	StartThreshold: 1.0,
	Name:           "Identify",
}

// InitIdentifyGraph initializes and returns a new graph.
func InitIdentifyGraph(gc services.GraphGenerator) *services.Graph {
	g := gc.NewGraph("RealtimeIdentify", &IdentifyStream{})

	permuteMul2 := PermuteMul2.DeepCopy()
	permute := graphs.Permute.DeepCopy()
	identifyMul2 := IdentifyMul2.DeepCopy()

	g.First(permuteMul2)
	g.Connect(permuteMul2, permute)
	g.Connect(permute, identifyMul2)
	g.Last(identifyMul2)

	return g
}
