////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package precomputation

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/gpumathsgo/cryptops"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/services"
)

// This file implements the Graph for the Precomputation Generate phase
// The Generate phase transforms first unpermuted internode keys
// and partial cypher texts into the data that the permute phase needs

// GenerateStream holds the inputs for the Generate operation
type GenerateStream struct {
	Grp *cyclic.Group

	// RNG
	RngStreamGen *fastRNG.StreamGenerator

	// Phase Keys
	R *cyclic.IntBuffer
	S *cyclic.IntBuffer
	U *cyclic.IntBuffer
	V *cyclic.IntBuffer

	// Share keys for each phase
	YR *cyclic.IntBuffer
	YS *cyclic.IntBuffer
	YU *cyclic.IntBuffer
	YV *cyclic.IntBuffer
}

// GetName returns the name of this op
func (gs *GenerateStream) GetName() string {
	return "PrecompGenerateStream"
}

// Link maps the local round data to the Generate Stream data structure (the input)
func (gs *GenerateStream) Link(grp *cyclic.Group, batchSize uint32, source ...interface{}) {
	roundBuffer := source[0].(*round.Buffer)
	rngStreamGen := source[1].(*fastRNG.StreamGenerator)

	gs.LinkGenerateStream(grp, batchSize, roundBuffer, rngStreamGen)
}

// LinkGenerateStream maps the local round data to the Generate Stream data structure (the input)
func (gs *GenerateStream) LinkGenerateStream(grp *cyclic.Group, batchSize uint32,
	roundBuffer *round.Buffer, rngStreamGen *fastRNG.StreamGenerator) {

	gs.Grp = grp

	gs.RngStreamGen = rngStreamGen

	// Phase keys
	gs.R = roundBuffer.R.GetSubBuffer(0, batchSize)
	gs.S = roundBuffer.S.GetSubBuffer(0, batchSize)
	gs.U = roundBuffer.U.GetSubBuffer(0, batchSize)
	gs.V = roundBuffer.V.GetSubBuffer(0, batchSize)

	// Share keys
	gs.YR = roundBuffer.Y_R.GetSubBuffer(0, batchSize)
	gs.YS = roundBuffer.Y_S.GetSubBuffer(0, batchSize)
	gs.YU = roundBuffer.Y_U.GetSubBuffer(0, batchSize)
	gs.YV = roundBuffer.Y_V.GetSubBuffer(0, batchSize)
}

type GenerateSubstreamInterface interface {
	GetGenerateSubStream() *GenerateStream
}

// GetGenerateSubStream implements reveal interface to return stream object
func (gs *GenerateStream) GetGenerateSubStream() *GenerateStream {
	return gs
}

// Input initializes stream inputs from slot received from IO
func (gs *GenerateStream) Input(index uint32, slot *mixmessages.Slot) error {
	if index >= uint32(gs.R.Len()) {
		return services.ErrOutsideOfBatch
	}
	return nil
}

// Output returns an empty cMixSlot message for IO
func (gs *GenerateStream) Output(index uint32) *mixmessages.Slot {
	return &mixmessages.Slot{}
}

// Generate implements cryptops.Generate for precomputation
var Generate = services.Module{
	// Generates key pairs R, S, U, and V
	Adapt: func(streamInput services.Stream, cryptop cryptops.Cryptop,
		chunk services.Chunk) error {
		gssi, ok := streamInput.(GenerateSubstreamInterface)
		generate, ok2 := cryptop.(cryptops.GeneratePrototype)

		if !ok || !ok2 {
			return services.InvalidTypeAssert
		}

		gs := gssi.GetGenerateSubStream()

		stream := gs.RngStreamGen.GetStream()

		for i := chunk.Begin(); i < chunk.End(); i++ {
			errors := []error{
				generate(gs.Grp, gs.R.Get(i), gs.YR.Get(i), stream),
				generate(gs.Grp, gs.S.Get(i), gs.YS.Get(i), stream),
				generate(gs.Grp, gs.U.Get(i), gs.YU.Get(i), stream),
				generate(gs.Grp, gs.V.Get(i), gs.YV.Get(i), stream),
			}
			for _, err := range errors {
				if err != nil {
					jww.FATAL.Panicf(err.Error())
				}
			}
		}
		stream.Close()

		return nil
	},
	Cryptop:    cryptops.Generate,
	NumThreads: services.AutoNumThreads,
	InputSize:  services.AutoInputSize,
	Name:       "Generate",
}

// InitGenerateGraph is called to initialize the Generate Graph. Conforms to Graph.Initialize function type
func InitGenerateGraph(gc services.GraphGenerator) *services.Graph {
	g := gc.NewGraph("PrecompGenerate", &GenerateStream{})

	generate := Generate.DeepCopy()

	g.First(generate)
	g.Last(generate)

	return g
}
