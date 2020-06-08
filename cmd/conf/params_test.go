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
		KeepBuffers:      true,
		RngScalingFactor: 10000,

		Node:          ExpectedNode,
		Database:      ExpectedDatabase,
		Gateway:       ExpectedGateway,
		Permissioning: ExpectedPermissioning,
		Metrics:       ExpectedMetrics,
		GraphGen:      ExpectedGraphGen,
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
		t.Fatalf("Failed in unmarshaling from viper object: %+v", err)
	}

	if !reflect.DeepEqual(expectedParams.Node, params.Node) {
		t.Errorf("Params node value does not match expected value\nActual: %v"+
			"\nExpected: %v", params.Node, expectedParams.Node)
	}

	if !reflect.DeepEqual(expectedParams.Database, params.Database) {
		t.Errorf("Params database value does not match expected value")
	}

	if expectedParams.KeepBuffers != params.KeepBuffers {
		t.Errorf("Params keepbuffers value does not match expected value")
	}

	if !reflect.DeepEqual(expectedParams.Gateway, params.Gateway) {
		t.Errorf("Params gateways value does not match expected value")
	}

	if !reflect.DeepEqual(expectedParams.Metrics, params.Metrics) {
		t.Errorf("Params metrics value does not match expected value")
	}

	if expectedParams.RngScalingFactor != params.RngScalingFactor {
		t.Errorf("RngScalingFactor value does not match expected value"+
			"\n\treceived:\t%v\n\texpected:\t%v",
			params.RngScalingFactor, expectedParams.RngScalingFactor)
	}

	if !reflect.DeepEqual(expectedParams.GraphGen, params.GraphGen) {
		t.Errorf("Graph generator values do not match expected values")
	}
}
