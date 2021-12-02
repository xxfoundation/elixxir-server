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
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/gpumathsgo"
	"gitlab.com/elixxir/gpumathsgo/cryptops"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/services"
)

type IdentifyStream struct {
	Grp        *cyclic.Group
	StreamPool *gpumaths.StreamPool

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

	var streamPool *gpumaths.StreamPool
	if len(source) >= 3 {
		// All arguments are being passed from the Link call, which should include the stream pool
		streamPool = source[2].(*gpumaths.StreamPool)
	}

	is.LinkIdentifyStreams(grp, batchSize, roundBuffer, streamPool,
		grp.NewIntBuffer(batchSize, grp.NewInt(1)),
		grp.NewIntBuffer(batchSize, grp.NewInt(1)),
		make([]*cyclic.Int, batchSize),
		make([]*cyclic.Int, batchSize))
}

// LinkRealtimePermuteStreams binds stream data.
func (is *IdentifyStream) LinkIdentifyStreams(grp *cyclic.Group, batchSize uint32, round *round.Buffer, pool *gpumaths.StreamPool,
	ecrPayloadA, ecrPayloadB *cyclic.IntBuffer, permPayloadA, permPayloadB []*cyclic.Int) {

	is.Grp = grp
	is.StreamPool = pool

	is.EcrPayloadA = ecrPayloadA
	is.EcrPayloadB = ecrPayloadB

	is.PayloadAPrecomputation = round.PayloadAPrecomputation.GetSubBuffer(0, batchSize)
	is.PayloadBPrecomputation = round.PayloadBPrecomputation.GetSubBuffer(0, batchSize)

	is.EcrPayloadAPermuted = permPayloadA
	is.EcrPayloadBPermuted = permPayloadB

	is.LinkRealtimePermuteStreams(grp, batchSize, round, pool,
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
		Index:    index,
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
			// Multiply encrypted payload A by its precomputation to decrypt it
			mul2(is.Grp, is.PayloadAPrecomputation.Get(i), is.EcrPayloadAPermuted[i])
			// Multiply encrypted payload B by its precomputation to decrypt it
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

// Module implementing cryptops.Mul2.
var IdentifyMul2Chunk = services.Module{
	// Multiplies in own Encrypted Keys and Partial Cypher Texts
	Adapt: func(stream services.Stream, cryptop cryptops.Cryptop, chunk services.Chunk) error {
		isi, ok := stream.(identifyStreamInterface)
		mul2Slice, ok2 := cryptop.(gpumaths.Mul2SlicePrototype)

		if !ok || !ok2 {
			return services.InvalidTypeAssert
		}

		is := isi.getIdentifyStream()

		//for i := chunk.Begin(); i < chunk.End(); i++ {
		pap := is.PayloadAPrecomputation.GetSubBuffer(chunk.Begin(), chunk.End())
		pbp := is.PayloadBPrecomputation.GetSubBuffer(chunk.Begin(), chunk.End())
		epap := is.EcrPayloadAPermuted[chunk.Begin():chunk.End()]
		epbp := is.EcrPayloadBPermuted[chunk.Begin():chunk.End()]

		// Multiply encrypted payload A by its precomputation to decrypt it
		err := mul2Slice(is.StreamPool, is.Grp, pap, epap, epap)
		if err != nil {
			return err
		}

		// Multiply encrypted payload B by its precomputation to decrypt it
		err = mul2Slice(is.StreamPool, is.Grp, pbp, epbp, epbp)
		if err != nil {
			return err
		}

		return nil
	},
	Cryptop:        gpumaths.Mul2Slice,
	NumThreads:     2,
	InputSize:      services.AutoInputSize,
	StartThreshold: 1.0,
	Name:           "Identify",
}

// InitIdentifyGraph initializes and returns a new graph.
func InitIdentifyGraph(gc services.GraphGenerator) *services.Graph {
	if viper.GetBool("useGPU") {
		jww.FATAL.Panicf("Using realtime identify graph running on CPU instead of equivalent GPU graph")
	}
	g := gc.NewGraph("RealtimeIdentify", &IdentifyStream{})

	permuteMul2 := PermuteMul2.DeepCopy()
	identifyMul2 := IdentifyMul2.DeepCopy()

	g.First(permuteMul2)
	g.Connect(permuteMul2, identifyMul2)
	g.Last(identifyMul2)

	return g
}

// InitIdentifyGraph initializes and returns a new graph.
func InitIdentifyGPUGraph(gc services.GraphGenerator) *services.Graph {
	if !viper.GetBool("useGPU") {
		jww.WARN.Printf("Using realtime identify graph running on GPU instead of equivalent CPU graph")
	}
	g := gc.NewGraph("RealtimeIdentifyGPU", &IdentifyStream{})

	permuteMul2 := PermuteMul2Chunk.DeepCopy()
	identifyMul2 := IdentifyMul2Chunk.DeepCopy()
	permuteMul2.InputSize = 32
	identifyMul2.InputSize = 32

	g.First(permuteMul2)
	g.Connect(permuteMul2, identifyMul2)
	g.Last(identifyMul2)

	return g
}
