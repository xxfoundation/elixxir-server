////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
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
	// Ignore extra permutations if there are more of them than slots for input/output
	// If there are more permutations than needed because an input size is larger,
	// those permutation should be a no-op and outside the batch, so ignoring them
	// is OK for buffers that are smaller
	for _, io := range IOs {
		n := len(io.Output)
		if io.Input.Len() < n {
			n = io.Input.Len()
		}
		for i := 0; i < n; i++ {
			io.Output[permutations[i]] = io.Input.Get(uint32(i))
		}
	}
}

// ModifyGraphGeneratorForPermute makes a copy of the graph generator
// where the OutputThreshold=1.0
func ModifyGraphGeneratorForPermute(gc services.GraphGenerator) services.GraphGenerator {
	return services.NewGraphGenerator(
		gc.GetMinInputSize(),
		gc.GetDefaultNumTh(),
		gc.GetOutputSize(),
		1.0,
	)
}
