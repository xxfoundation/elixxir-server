///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package services

import "testing"

//Tests that thresholding works in a variety of cases
func TestThreshold(t *testing.T) {
	testThresholhelper(25, 1.0, 25, t)
	testThresholhelper(25, 0.5, 12, t)
	testThresholhelper(25, 0, 0, t)

	testThresholhelper(24, 1.0, 24, t)
	testThresholhelper(24, 0.5, 12, t)
	testThresholhelper(24, 0, 0, t)
}

func testThresholhelper(batchSize uint32, thresh float32, expected uint32,
	t *testing.T) {
	th := threshold(batchSize, thresh)

	if th != expected {
		t.Errorf("Thresholding: Incorrect for batchsize %v at "+
			"threshold: %v; Expected; %v, Received: %v", batchSize, thresh,
			expected, th)
	}
}
