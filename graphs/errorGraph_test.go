////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package graphs

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"testing"
)

func TestErrorStream_GetName(t *testing.T) {
	testES := ErrorStream{}
	exptectedName := "ErrorStream"
	testName := testES.GetName()

	if testName != exptectedName {
		t.Errorf("GetName() did not return the expected name."+
			"\n\texpected: %v\n\treceived: %v", exptectedName, testName)
	}
}

func TestErrorStream_Output(t *testing.T) {
	testES := ErrorStream{}
	expectedOutput := &mixmessages.Slot{}
	testOutput := testES.Output(5)

	if expectedOutput.String() != testOutput.String() {
		t.Errorf("Output() did not return the expected output."+
			"\n\texpected: %#v\n\treceived: %#v",
			expectedOutput.String(), testOutput.String())
	}
}
