////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package precomputation

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

// This file implements the Graph for the Precomputation Reveal phase.
// The Reveal phase removes cypher keys from both payload's cypher texts,
// revealing the private keys for the round.

// RevealStream holds data containing private key from encrypt and inputs used by strip
type RevealStream struct {
	Grp        *cyclic.Group
	StreamPool *gpumaths.StreamPool

	//Link to round object
	Z *cyclic.Int

	// Unique to stream
	CypherPayloadA *cyclic.IntBuffer
	CypherPayloadB *cyclic.IntBuffer
}

// GetName returns stream name
func (s *RevealStream) GetName() string {
	return "PrecompRevealStream"
}

// Link binds stream to local state objects in round
func (s *RevealStream) Link(grp *cyclic.Group, batchSize uint32, source ...interface{}) {
	roundBuffer := source[0].(*round.Buffer)
	var streamPool *gpumaths.StreamPool
	if len(source) >= 3 {
		// All arguments are being passed from the Link call, which should include the stream pool
		streamPool = source[2].(*gpumaths.StreamPool)
	}

	s.LinkRevealStream(grp, roundBuffer, streamPool,
		grp.NewIntBuffer(batchSize, grp.NewInt(1)),
		grp.NewIntBuffer(batchSize, grp.NewInt(1)))
}

// LinkRevealStream binds stream to local state objects in round
func (s *RevealStream) LinkRevealStream(grp *cyclic.Group, roundBuffer *round.Buffer, streamPool *gpumaths.StreamPool, CypherPayloadA, CypherPayloadB *cyclic.IntBuffer) {
	s.Grp = grp
	s.StreamPool = streamPool

	s.Z = roundBuffer.Z

	s.CypherPayloadA = CypherPayloadA
	s.CypherPayloadB = CypherPayloadB
}

// Input initializes stream inputs from slot received from IO
func (s *RevealStream) Input(index uint32, slot *mixmessages.Slot) error {

	if index >= uint32(s.CypherPayloadA.Len()) {
		return services.ErrOutsideOfBatch
	}

	if !s.Grp.BytesInside(slot.PartialPayloadACypherText, slot.PartialPayloadBCypherText) {
		return services.ErrOutsideOfGroup
	}

	s.Grp.SetBytes(s.CypherPayloadA.Get(index), slot.PartialPayloadACypherText)
	s.Grp.SetBytes(s.CypherPayloadB.Get(index), slot.PartialPayloadBCypherText)
	return nil
}

// Output a cmix slot message for IO
func (s *RevealStream) Output(index uint32) *mixmessages.Slot {
	return &mixmessages.Slot{
		Index:                     index,
		PartialPayloadACypherText: s.CypherPayloadA.Get(index).Bytes(),
		PartialPayloadBCypherText: s.CypherPayloadB.Get(index).Bytes(),
	}
}

type revealSubstreamInterface interface {
	getRevealSubStream() *RevealStream
}

// getSubStream implements reveal interface to return stream object
func (s *RevealStream) getRevealSubStream() *RevealStream {
	return s
}

// RevealRootCoprime is a CPU module in precomputation reveal implementing cryptops.RootCoprimePrototype
var RevealRootCoprime = services.Module{
	// Runs root coprime for cypher texts
	Adapt: func(streamInput services.Stream, cryptop cryptops.Cryptop, chunk services.Chunk) error {
		s, ok := streamInput.(revealSubstreamInterface)
		rootCoprime, ok2 := cryptop.(cryptops.RootCoprimePrototype)

		if !ok || !ok2 {
			return services.InvalidTypeAssert
		}

		rs := s.getRevealSubStream()
		tmp := rs.Grp.NewMaxInt()

		for i := chunk.Begin(); i < chunk.End(); i++ {
			// Execute rootCoprime on the keys for the first payload
			// Eq 15.11 Root by cypher key to remove one layer of homomorphic
			// encryption from partially encrypted payload A cypher text.

			rootCoprime(rs.Grp, rs.CypherPayloadA.Get(i), rs.Z, tmp)
			rs.Grp.Set(rs.CypherPayloadA.Get(i), tmp)

			// Execute rootCoprime on the keys for the second payload
			// Eq 15.13 Root by cypher key to remove one layer of homomorphic
			// encryption from partially encrypted payload B cypher text.
			rootCoprime(rs.Grp, rs.CypherPayloadB.Get(i), rs.Z, tmp)
			rs.Grp.Set(rs.CypherPayloadB.Get(i), tmp)
		}
		return nil
	},
	Cryptop:    cryptops.RootCoprime,
	NumThreads: services.AutoNumThreads,
	InputSize:  services.AutoInputSize,
	Name:       "RevealRootCoprime",
}

// RevealRootCoprimeChunk is a GPU module in precomputation reveal implementing cryptops.RootCoprimePrototype
var RevealRootCoprimeChunk = services.Module{
	Adapt: func(streamInput services.Stream, cryptop cryptops.Cryptop, chunk services.Chunk) error {
		rssi, ok := streamInput.(revealSubstreamInterface)
		rc, ok2 := cryptop.(gpumaths.RevealChunkPrototype)
		if !ok || !ok2 {
			return services.InvalidTypeAssert
		}
		rs := rssi.getRevealSubStream()
		gpuStreams := rs.StreamPool
		cpa := rs.CypherPayloadA.GetSubBuffer(chunk.Begin(), chunk.End())
		err := rc(gpuStreams, rs.Grp, rs.Z, cpa, cpa)
		if err != nil {
			return err
		}

		cpb := rs.CypherPayloadB.GetSubBuffer(chunk.Begin(), chunk.End())
		err = rc(gpuStreams, rs.Grp, rs.Z, cpb, cpb)
		if err != nil {
			return err
		}

		return nil
	},
	Cryptop:    gpumaths.RevealChunk,
	Name:       "RevealRootCoprimeGPU",
	NumThreads: 2,
}

// InitRevealGraph called to initialize the CPU Graph. Conforms to graphs.Initialize function type
func InitRevealGraph(gc services.GraphGenerator) *services.Graph {
	if viper.GetBool("useGPU") {
		jww.FATAL.Panicf("Using precomp reveal graph running on CPU instead of equivalent GPU graph")
	}
	graph := gc.NewGraph("PrecompReveal", &RevealStream{})

	revealRootCoprime := RevealRootCoprime.DeepCopy()

	graph.First(revealRootCoprime)
	graph.Last(revealRootCoprime)

	return graph
}

// InitRevealGPUGraph called to initialize the GPU Graph. Conforms to graphs.Initialize function type
func InitRevealGPUGraph(gc services.GraphGenerator) *services.Graph {
	if !viper.GetBool("useGPU") {
		jww.WARN.Printf("Using precomp reveal graph running on GPU instead of equivalent CPU graph")
	}
	g := gc.NewGraph("PrecompRevealGPU", &RevealStream{})

	revealRootCoprimeChunk := RevealRootCoprimeChunk.DeepCopy()

	g.First(revealRootCoprimeChunk)
	g.Last(revealRootCoprimeChunk)

	return g
}
