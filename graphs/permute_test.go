////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package graphs

import (
	"gitlab.com/elixxir/server/services"
	"runtime"
	"testing"
)

func TestModifyGraphGeneratorForPermute(t *testing.T) {
	gc := services.NewGraphGenerator(4, func(err error) { return }, uint8(runtime.NumCPU()), 1, 0)

	gcPermute := ModifyGraphGeneratorForPermute(gc)

	if gcPermute.GetOutputSize() != gc.GetOutputSize() {
		t.Errorf("ModifyGraphGeneratorForPermute: Output not copied correctly, "+
			"Expected: %v, Recieved: %v", gc.GetOutputSize(), gcPermute.GetOutputSize())
	}

	if gcPermute.GetDefaultNumTh() != gc.GetDefaultNumTh() {
		t.Errorf("ModifyGraphGeneratorForPermute: DefaultNumThreads not copied correctly, "+
			"Expected: %v, Recieved: %v", gc.GetDefaultNumTh(), gcPermute.GetDefaultNumTh())
	}

	if gcPermute.GetMinInputSize() != gc.GetMinInputSize() {
		t.Errorf("ModifyGraphGeneratorForPermute: MinInputSize not copied correctly, "+
			"Expected: %v, Recieved: %v", gc.GetMinInputSize(), gcPermute.GetMinInputSize())
	}

	//test that the function on the output is the same as the input

	if gc.GetOutputThreshold() != 1.0 {
		t.Errorf("ModifyGraphGeneratorForPermute: OutputThreshold not set correctly, "+
			"Expected: %v, Recieved: %v", 1.0, gcPermute.GetOutputThreshold())
	}
}
