////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package graphs

import (
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/services"
)

// PermuteIO used to convert input and output when streams are linked
type PermuteIO struct {
	Input  *cyclic.IntBuffer
	Output []*cyclic.Int
}

// PermuteSubStream is used to store input and outputs slices for permutation
type PermuteSubStream struct {
	// Populate during Link
	permutations []uint32
	inputs       []*cyclic.IntBuffer

	// New variable, created during Link
	outputs [][]*cyclic.Int
}

// LinkStreams sets array of permutations slice references and appends each permute io from list into substream
func (pss *PermuteSubStream) LinkStreams(expandedBatchSize uint32, permutation []uint32, ioLst ...PermuteIO) {

	pss.permutations = permutation
	for _, io := range ioLst {
		pss.inputs = append(pss.inputs, io.Input)
		pss.outputs = append(pss.outputs, io.Output)
	}
}

type permuteSubStreamInterface interface {
	getSubStream() *PermuteSubStream
}

func (pss *PermuteSubStream) getSubStream() *PermuteSubStream {
	return pss
}

// Permute module implements slot permutations on Adapt and conforms to module interface using a dummy cryptop
var Permute = services.Module{
	Adapt: func(stream services.Stream, cryptop cryptops.Cryptop, chunk services.Chunk) error {
		ps, ok := stream.(permuteSubStreamInterface)

		if !ok {
			return services.InvalidTypeAssert
		}

		pss := ps.getSubStream()

		// do the permutations for the requested chunk
		// rEaL iNtEnSe cRyPtOgRaPhy goInG oN hERe
		for itr := range pss.inputs {
			for j := chunk.Begin(); j < chunk.End(); j++ {
				pss.outputs[itr][pss.permutations[j]] = pss.inputs[itr].Get(j)
			}
		}
		return nil
	},
	Cryptop:        permuteDummyCryptop,
	InputSize:      services.AUTO_INPUTSIZE,
	StartThreshold: 0,
	Name:           "Permute",
	NumThreads:     4,
}

/* Dummy cryptop for testing*/
type permuteDummyCryptopPrototype func()

var permuteDummyCryptop permuteDummyCryptopPrototype = func() { return }

// Returns the name for debugging
func (permuteDummyCryptopPrototype) GetName() string {
	return "Permute Dummy Cryptop"
}

// Returns the input size, used in safety checks
func (permuteDummyCryptopPrototype) GetInputSize() uint32 {
	return 1
}
