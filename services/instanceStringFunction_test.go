///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package services

import "testing"

// Tests that the string outputted contains the expected results given the input.
func TestNameStringer(t *testing.T) {
	str := NameStringer("1.1.1.1", 0, 1)

	if str != "1.1.1.1 - (1/1)" {
		t.Logf("Name Stringer failed to return expected output: %s", str)
		t.Fail()
	}
}
