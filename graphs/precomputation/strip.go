////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package precomputation

// Precomp Strip

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

// This file implements the Graph for the Precomputation Strip phase.
// Strip phase inverts the Round Private Keys and removes the
// homomorphic encryption from the encrypted keys, revealing completed
// precomputation

// StripStream holds data containing private key from encrypt and inputs used by strip
type StripStream struct {
	Grp        *cyclic.Group
	StreamPool *gpumaths.StreamPool

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

// Link binds stream to local state objects in round
func (ss *StripStream) Link(grp *cyclic.Group, batchSize uint32,
	source ...interface{}) {
	roundBuffer := source[0].(*round.Buffer)

	var streamPool *gpumaths.StreamPool
	if len(source) >= 3 {
		// All arguments are being passed from the Link call, which should include the stream pool
		streamPool = source[2].(*gpumaths.StreamPool)
	}

	ss.LinkStripStream(grp, batchSize, roundBuffer, streamPool,
		grp.NewIntBuffer(batchSize, grp.NewInt(1)),
		grp.NewIntBuffer(batchSize, grp.NewInt(1)))

}

// LinkStripStream binds stream to local state objects in round
func (ss *StripStream) LinkStripStream(grp *cyclic.Group,
	batchSize uint32, roundBuf *round.Buffer, pool *gpumaths.StreamPool,
	cypherPayloadA, cypherPayloadB *cyclic.IntBuffer) {

	ss.Grp = grp
	ss.StreamPool = pool

	ss.PayloadAPrecomputation = roundBuf.PayloadAPrecomputation.GetSubBuffer(
		0, batchSize)
	ss.PayloadBPrecomputation = roundBuf.PayloadBPrecomputation.GetSubBuffer(
		0, batchSize)

	ss.EncryptedPayloadAPrecomputation = roundBuf.PermutedPayloadAKeys
	ss.EncryptedPayloadBPrecomputation = roundBuf.PermutedPayloadBKeys

	ss.CypherPayloadA = cypherPayloadA
	ss.CypherPayloadB = cypherPayloadB

	ss.RevealStream.LinkRevealStream(grp, roundBuf, pool, ss.CypherPayloadA, ss.CypherPayloadB)
}

type stripSubstreamInterface interface {
	GetStripSubStream() *StripStream
}

// GetStripSubStream implements reveal interface to return stream object
func (ss *StripStream) GetStripSubStream() *StripStream {
	return ss
}

// Input initializes stream inputs from slot received from IO
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

// Output a cmix slot message for IO
func (ss *StripStream) Output(index uint32) *mixmessages.Slot {
	return &mixmessages.Slot{
		Index: index,
		EncryptedPayloadAKeys: ss.PayloadAPrecomputation.Get(
			index).Bytes(),
		EncryptedPayloadBKeys: ss.PayloadBPrecomputation.Get(index).Bytes(),
	}
}

// StripInverse is a module in precomputation strip implementing cryptops.Inverse
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
	NumThreads: services.AutoNumThreads,
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

// InitStripGraph is called to initialize the CPU Graph. Conforms to Graph.Initialize function type
func InitStripGraph(gc services.GraphGenerator) *services.Graph {
	if viper.GetBool("useGPU") {
		jww.FATAL.Panicf("Using precomp strip graph running on CPU instead of equivalent GPU graph")
	}
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

// InitStripGPUGraph is called to initialize the GPU Graph. Conforms to Graph.Initialize function type
func InitStripGPUGraph(gc services.GraphGenerator) *services.Graph {
	if !viper.GetBool("useGPU") {
		jww.WARN.Printf("Using precomp strip graph running on GPU instead of equivalent CPU graph")
	}
	graph := gc.NewGraph("PrecompStripGPU", &StripStream{})

	// GPU library can do all operations for Strip in one kernel,
	// to avoid uploading and downloading excessively
	// or having to build an abstraction over bindings
	// between dispatcher graphs and CUDA graphs
	// for running separate CUDA kernels without additional
	// overhead.
	// For some reason, the strip kernel doesn't work correctly
	// in the real rounds. So for now we're using reveal on GPU
	// and do the rest on CPU.
	reveal := RevealRootCoprimeChunk.DeepCopy()
	stripInverse := StripInverse.DeepCopy()
	stripMul2 := StripMul2.DeepCopy()

	graph.First(reveal)
	graph.Connect(reveal, stripInverse)
	graph.Connect(stripInverse, stripMul2)
	graph.Last(stripMul2)

	return graph
}
