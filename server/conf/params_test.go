////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

import (
	"github.com/spf13/viper"
	"reflect"
	"testing"
)

func TestNewParams_ReturnsParamsWhenGivenValidViper(t *testing.T) {

	expectedParams := Params{
		Database: ExpectedDB,
		Groups:   ExpectedGroups,
		Paths:    ExpectedPaths,
		Servers:  []string{"127.0.0.1:80", "127.0.0.1:80", "127.0.0.1:80"},
		SkipReg:  true,
		NodeID:   uint64(100),
	}

	vip := viper.New()
	vip.AddConfigPath(".")
	vip.SetConfigFile("params.yaml")
	vip.Set("NodeID", uint64(100))

	err := vip.ReadInConfig()

	if err != nil {
		t.Errorf("Failed to read in params.yaml into viper")
	}

	params, err := NewParams(vip)

	if err != nil {
		t.Errorf("Failed in unmarshaling from viper object")
	}

	if !reflect.DeepEqual(expectedParams.Servers, params.Servers) {
		t.Errorf("Servers value does not match expected value")
	}

	if !reflect.DeepEqual(expectedParams.NodeID, params.NodeID) {
		t.Errorf("NodeId value does not match expected value")
	}

	if !reflect.DeepEqual(expectedParams.SkipReg, params.SkipReg) {
		t.Errorf("SkipReg value does not match expected value")
	}

	if params.Groups.E2E.GetFingerprint() != ExpectedGroups.E2E.GetFingerprint() {
		t.Errorf("E2E object did not match expected values for E2E")
	}
	if params.Groups.CMix.GetFingerprint() != ExpectedGroups.CMix.GetFingerprint() {
		t.Errorf("CMIX object did not match expected values for CMIX")
	}

	if !reflect.DeepEqual(expectedParams.Paths, params.Paths) {
		t.Errorf("Paths value does not match expected value")
	}

	if !reflect.DeepEqual(expectedParams.Database, params.Database) {
		t.Errorf("v value does not match expected value")
	}

}
