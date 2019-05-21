////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

import "testing"

func TestNewParams_ErrorOnEmptyDB(t *testing.T) {
	params := Params{}
	var err error

	err = params.SetDB("", "", "", nil)
	if err != nil {
		t.Errorf("No bueno")
	}

	err = params.SetGroups("", "", "", nil)
	if err != nil {
		t.Errorf("No bueno")
	}

	err = params.SetPaths("", "", "", nil)
	if err != nil {
		t.Errorf("No bueno")
	}

	err = params.SetContext("", "", "", nil)
	if err != nil {
		t.Errorf("No bueno")
	}

	err = params.SetRegistry("", "", "", nil)
	if err != nil {
		t.Errorf("No bueno")
	}

}
