///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////
package io

import (
	"bytes"
	"git.xx.network/elixxir/primitives/current"
	"git.xx.network/elixxir/server/testUtil"
	"testing"
)

func TestGetNdf(t *testing.T) {
	instance, _, _ := createMockInstance(t, 0, current.REALTIME)

	receivedNdf, err := GetNdf(instance)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expectedNdf, _ := testUtil.NDF.Marshal()

	if !bytes.Equal(receivedNdf, expectedNdf) {
		t.Errorf("Did not get expected result!"+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", expectedNdf, receivedNdf)
	}
}
