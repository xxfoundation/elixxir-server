////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package phase

import "testing"

//Tests that the number of phases and the number of phase strings are the same
func TestPhase(t *testing.T) {
	if len(typeStrings) != int(NumPhases) {
		t.Errorf("Number of phase strings (%v) not equal to number of phases (%v)",
			len(typeStrings), int(NumPhases))
	}
}

//Tests that the correct phases are returned
func TestPhase_String(t *testing.T) {
	for i := Type(0); i < NumPhases; i++ {
		if i.String() != typeStrings[i] {
			t.Errorf("phase.String does not outpur the correct result for phase %s",
				typeStrings[i])
		}
	}
}
