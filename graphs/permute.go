////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package graphs

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/services"
)

// PermuteIO used to convert input and output when streams are linked
type PermuteIO struct {
	Input  *cyclic.IntBuffer
	Output []*cyclic.Int
}

// PrecanPermute connects the input intBuffers to the output int slices.
// for use inside of a link function
func PrecanPermute(permutations []uint32, IOs ...PermuteIO) {
	for index, permutation := range permutations {
		for _, io := range IOs {
			io.Output[permutation] = io.Input.Get(uint32(index))
		}
	}
}

// ModifyGraphGeneratorForPermute makes a copy of the graph generator
// where the OutputThreshold=1.0
func ModifyGraphGeneratorForPermute(gc services.GraphGenerator) services.GraphGenerator {
	return services.NewGraphGenerator(
		gc.GetMinInputSize(),
		gc.GetErrorHandler(),
		gc.GetDefaultNumTh(),
		gc.GetOutputSize(),
		1.0,
	)
}
