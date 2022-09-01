//+build gpu

////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package main

import (
	"gitlab.com/elixxir/server/cmd"
	"testing"
)

func Test_MultiInstance_N3_B32_GPU(t *testing.T) {
	batchSize := 32
	if cmd.BatchSizeGPUTest != 0 {
		batchSize = cmd.BatchSizeGPUTest
	}

	elapsed := MultiInstanceTest(3, batchSize, makeMultiInstanceGroup(), true, false, t)

	t.Logf("Computational elapsed time for 3 Node, batch size %d, GPU multi-"+
		"instance test: %s", cmd.BatchSizeGPUTest, elapsed)
}
