////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package precomputation

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/node"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/services"
)

// This file implements the Graph for the Precomputation Generate phase
// Generate phase transforms first unpermuted internode keys
// and partial cypher texts into the data that the permute phase needs

// GenerateStream holds the inputs for the Generate operation
type GenerateStream struct {
	Grp *cyclic.Group

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
func (s *GenerateStream) GetName() string {
	return "PrecompGenerateStream"
}

// Link maps the round data to the Generate Stream data structure (the input)
func (s *GenerateStream) Link(grp *cyclic.Group, batchSize uint32, source interface{}) {
	roundBuffer := source.(*round.Buffer)

	s.Grp = grp

	// Phase keys
	s.R = roundBuffer.R.GetSubBuffer(0, batchSize)
	s.S = roundBuffer.S.GetSubBuffer(0, batchSize)
	s.U = roundBuffer.U.GetSubBuffer(0, batchSize)
	s.V = roundBuffer.V.GetSubBuffer(0, batchSize)

	// Share keys
	s.YR = roundBuffer.Y_R.GetSubBuffer(0, batchSize)
	s.YS = roundBuffer.Y_S.GetSubBuffer(0, batchSize)
	s.YU = roundBuffer.Y_U.GetSubBuffer(0, batchSize)
	s.YV = roundBuffer.Y_V.GetSubBuffer(0, batchSize)
}

// Input function pulls things from the mixmessage
func (s *GenerateStream) Input(index uint32, slot *mixmessages.Slot) error {
	if index >= uint32(s.R.Len()) {
		return node.ErrOutsideOfBatch
	}
	return nil
}

// Output returns an empty cMixSlot message
func (s *GenerateStream) Output(index uint32) *mixmessages.Slot {
	return &mixmessages.Slot{}
}

// Generate does precomputation for implementing cryptops.Generate
var Generate = services.Module{
	// Multiplies in own Encrypted Keys and Partial Cypher Texts
	Adapt: func(streamInput services.Stream, cryptop cryptops.Cryptop,
		chunk services.Chunk) error {
		s, ok := streamInput.(*GenerateStream)
		generate, ok2 := cryptop.(cryptops.GeneratePrototype)

		if !ok || !ok2 {
			return services.InvalidTypeAssert
		}

		rng := csprng.NewSystemRNG()

		for i := chunk.Begin(); i < chunk.End(); i++ {
			errors := []error{
				generate(s.Grp, s.R.Get(i), s.YR.Get(i), rng),
				generate(s.Grp, s.S.Get(i), s.YS.Get(i), rng),
				generate(s.Grp, s.U.Get(i), s.YU.Get(i), rng),
				generate(s.Grp, s.V.Get(i), s.YV.Get(i), rng),
			}
			for _, err := range errors {
				if err != nil {
					jww.CRITICAL.Panicf(err.Error())
				}
			}
		}
		return nil
	},
	Cryptop:    cryptops.Generate,
	NumThreads: 5,
	InputSize:  services.AUTO_INPUTSIZE,
	Name:       "Generate",
}

// InitGenerateGraph initializes a new generate graph
func InitGenerateGraph(gc services.GraphGenerator) *services.Graph {
	g := gc.NewGraph("PrecompGenerate", &GenerateStream{})

	generate := Generate.DeepCopy()

	g.First(generate)
	g.Last(generate)

	return g
}
