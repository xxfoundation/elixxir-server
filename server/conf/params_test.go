////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

import "testing"

func TestNewParams_EnablesSetDB(t *testing.T) {

	params := NewParams()

	err := params.SetDB(ValidDBName, ValidUserName, ValidPassword, ValidAddresses)

	if err != nil {
		t.Errorf("NewParams failed to enable SetDB")
	}

}

func TestNewParams_EnablesSetGroups(t *testing.T) {

	params := NewParams()

	err := params.SetGroups(ValidCMixGrp, ValidE2EGrp)

	if err != nil {
		t.Errorf("NewParams failed to enable SetGroups")
	}

}

func TestNewParams_EnablesSetContext(t *testing.T) {

	params := NewParams()

	err := params.SetContext(ValidSevers, ValidNodeId)

	if err != nil {
		t.Errorf("NewParams failed to enable SetContext")
	}

}

func TestNewParams_EnablesSetGroups(t *testing.T) {

	params := NewParams()

	err := params.SetPaths(ValidCertPath, ValidKeyPath, ValidLogPath)

	if err != nil {
		t.Errorf("NewParams failed to enable SetPaths")
	}

}

func TestNewParams_EnablesSetRegistry(t *testing.T) {

	params := NewParams()

	err := params.SetRegistry(true)

	if err != nil {
		t.Errorf("NewParams failed to enable SetRegistry")
	}

}
