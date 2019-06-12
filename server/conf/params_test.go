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
		NodeAddresses: []string{"127.0.0.1:80", "127.0.0.1:80",
			"127.0.0.1:80"},
		Gateways: []string{"127.0.0.1:80", "127.0.0.1:80", "127.0.0.1:80"},
		SkipReg:  true,
		Index:    1,
		NodeIDs: []string{
			"pneumonoultramicroscopicsilicovolcanoconios=",
			"pneumonoultramicroscopicsilicovolcanoconios=",
			"pneumonoultramicroscopicsilicovolcanoconios=",
		},
		Batch: 20,
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
		t.Fatalf("Failed in unmarshaling from viper object")
	}

	if !reflect.DeepEqual(expectedParams.NodeAddresses, params.NodeAddresses) {
		t.Errorf("Server addresses value does not match expected value")
	}

	if !reflect.DeepEqual(expectedParams.Batch, params.Batch) {
		t.Errorf("Batch size value does not match expected value")
	}

	if !reflect.DeepEqual(expectedParams.Gateways, params.Gateways) {
		t.Errorf("Gateways value does not match expected value")
	}

	if !reflect.DeepEqual(expectedParams.Index, params.Index) {
		t.Errorf("NodeIndex value does not match expected value")
	}

	if !reflect.DeepEqual(expectedParams.NodeIDs, params.NodeIDs) {
		t.Errorf("NodeIDs value does not match expected value")
	}

	if !reflect.DeepEqual(expectedParams.SkipReg, params.SkipReg) {
		t.Errorf("SkipReg value does not match expected value")
	}

	if params.Groups.E2E.GetFingerprint() != ExpectedGroups.E2E.GetFingerprint() {
		t.Errorf("Groups object did not match expected values for E2E")
	}
	if params.Groups.CMix.GetFingerprint() != ExpectedGroups.CMix.GetFingerprint() {
		t.Errorf("Groups object did not match expected values for CMIX")
	}

	if !reflect.DeepEqual(expectedParams.Paths, params.Paths) {
		t.Errorf("Paths value does not match expected value")
	}

	if !reflect.DeepEqual(expectedParams.Database, params.Database) {
		t.Errorf("v value does not match expected value")
	}

}
