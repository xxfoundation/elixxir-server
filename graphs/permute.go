///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

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
	for _, io := range IOs {
		for i := range io.Output {
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
