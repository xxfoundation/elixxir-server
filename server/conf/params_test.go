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
		Node:          ExpectedNode,
		Database:      ExpectedDatabase,
		Gateways:      ExpectedGateways,
		Permissioning: ExpectedPermissioning,
		Global:        ExpectedGlobal,
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
		t.Errorf("Params value does not match expected value")
	}

	if !reflect.DeepEqual(expectedParams.Database, params.Database) {
		t.Errorf("Params database value does not match expected value")
	}

	if !reflect.DeepEqual(expectedParams.Global, params.Global) {
		t.Errorf("Params global value does not match expected value")
	}

	if !reflect.DeepEqual(expectedParams.Permissioning, params.Permissioning) {
		t.Errorf("Params node value does not match expected value")
	}

	if !reflect.DeepEqual(expectedParams.Gateways, params.Gateways) {
		t.Errorf("Params node value does not match expected value")
	}

}
