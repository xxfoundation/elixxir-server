///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package realtime

import (
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/gpumathsgo"
	"gitlab.com/elixxir/server/graphs"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/services"
)

type PermuteStream struct {
	Grp        *cyclic.Group
	StreamPool *gpumaths.StreamPool

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

	var streamPool *gpumaths.StreamPool
	if len(source) >= 4 {
		// All arguments are being passed from the Link call, which should include the stream pool
		streamPool = source[3].(*gpumaths.StreamPool)
	}

	ps.LinkRealtimePermuteStreams(grp, batchSize, roundBuffer, streamPool,
		grp.NewIntBuffer(batchSize, grp.NewInt(1)),
		grp.NewIntBuffer(batchSize, grp.NewInt(1)),
		make([]*cyclic.Int, batchSize),
		make([]*cyclic.Int, batchSize))
}

// LinkPermuteStreams binds stream data.
func (ps *PermuteStream) LinkRealtimePermuteStreams(grp *cyclic.Group,
	batchSize uint32, roundBuffer *round.Buffer, pool *gpumaths.StreamPool, msg, ad *cyclic.IntBuffer, msgPerm,
	adPerm []*cyclic.Int) {
	ps.Grp = grp
	ps.StreamPool = pool

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

// Module implementing gpumaths.Mul2Chunk
var PermuteMul2Chunk = services.Module{
	Adapt: func(stream services.Stream, cryptop cryptops.Cryptop,
		chunk services.Chunk) error {
		psi, ok1 := stream.(permuteSubStreamInterface)
		mul2Chunk, ok2 := cryptop.(gpumaths.Mul2ChunkPrototype)

		if !ok1 || !ok2 {
			return services.InvalidTypeAssert
		}

		ps := psi.getPermuteSubStream()

		pool := ps.StreamPool
		S := ps.S.GetSubBuffer(chunk.Begin(), chunk.End())
		V := ps.V.GetSubBuffer(chunk.Begin(), chunk.End())
		epa := ps.EcrPayloadA.GetSubBuffer(chunk.Begin(), chunk.End())
		epb := ps.EcrPayloadB.GetSubBuffer(chunk.Begin(), chunk.End())
		err := mul2Chunk(pool, ps.Grp, epa, S, epa)
		if err != nil {
			return err
		}
		err = mul2Chunk(pool, ps.Grp, epb, V, epb)
		if err != nil {
			return err
		}

		return nil
	},
	Cryptop:    gpumaths.Mul2Chunk,
	InputSize:  services.AutoInputSize,
	Name:       "PermuteRealtimeGPU",
	NumThreads: 2,
}

// InitPermuteGraph initializes and returns a new graph.
func InitPermuteGraph(gc services.GraphGenerator) *services.Graph {
	if viper.GetBool("useGpu") {
		jww.FATAL.Panicf("Using realtime permute graph running on CPU instead of equivalent GPU graph")
	}
	gcPermute := graphs.ModifyGraphGeneratorForPermute(gc)
	g := gcPermute.NewGraph("RealtimePermute", &PermuteStream{})

	mul2 := PermuteMul2.DeepCopy()

	g.First(mul2)
	g.Last(mul2)

	return g
}

// InitPermuteGPUGraph initializes a graph that uses the GPU
func InitPermuteGPUGraph(gc services.GraphGenerator) *services.Graph {
	if !viper.GetBool("useGpu") {
		jww.WARN.Printf("Using realtime permute graph running on GPU instead of equivalent CPU graph")
	}
	gcPermute := graphs.ModifyGraphGeneratorForPermute(gc)
	g := gcPermute.NewGraph("RealtimePermuteGPU", &PermuteStream{})

	mul2 := PermuteMul2Chunk.DeepCopy()
	mul2.InputSize = 32

	g.First(mul2)
	g.Last(mul2)

	return g
}
