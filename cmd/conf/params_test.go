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
		Index:       5,
		Batch:       uint32(20),
		SkipReg:     true,
		Verbose:     true,
		KeepBuffers: true,
		Groups:      ExpectedGroups,

		Node:          ExpectedNode,
		Database:      ExpectedDatabase,
		Gateways:      ExpectedGateways,
		Permissioning: ExpectedPermissioning,
		Metrics:       ExpectedMetrics,
	}

	vip := viper.New()
	vip.AddConfigPath(".")
	vip.SetConfigFile("params.yaml")

	err := vip.ReadInConfig()

	if err != nil {
		t.Errorf("Failed to read in params.yaml into viper")
	}

	params, err := NewParams(vip)

	if err != nil {
		t.Fatalf("Failed in unmarshaling from viper object")
	}

	if !reflect.DeepEqual(expectedParams.Node, params.Node) {
		t.Errorf("Params node value does not match expected value\nActual: %v"+
			"\nExpected: %v", params.Node, expectedParams.Node)
	}

	if !reflect.DeepEqual(expectedParams.Database, params.Database) {
		t.Errorf("Params database value does not match expected value")
	}

	if !reflect.DeepEqual(expectedParams.Groups, params.Groups) {
		t.Errorf("Params group value does not match expected value")
	}

	if expectedParams.Index != 5 {
		t.Errorf("Params index value does not match expected value")
	}

	if expectedParams.Batch != 20 {
		t.Errorf("Params batch value does not match expected value")
	}

	if expectedParams.SkipReg != true {
		t.Errorf("Params skipreg value does not match expected value")
	}

	if expectedParams.KeepBuffers != true {
		t.Errorf("Params keepbuffers value does not match expected value")
	}

	if expectedParams.Verbose != true {
		t.Errorf("Params verbose value does not match expected value")
	}

	if !reflect.DeepEqual(expectedParams.Gateways, params.Gateways) {
		t.Errorf("Params gateways value does not match expected value")
	}

	if !reflect.DeepEqual(expectedParams.Metrics, params.Metrics) {
		t.Errorf("Params metrics value does not match expected value")
	}
}
