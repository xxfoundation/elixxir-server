////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

import (
	"gitlab.com/elixxir/primitives/utils"
	"gopkg.in/yaml.v2"
	"reflect"
	"testing"
)

var ExpectedGateway = Gateway{
	Paths: Paths{
		Cert: "~/.elixxir/gateway.crt",
		Key:  "",
		Log:  "",
	},
	Address: "127.0.0.1:80",
}

// This test checks that unmarshalling the params.yaml file
// has the expected Gateways object.
func TestGateways_UnmarshallingFileEqualsExpected(t *testing.T) {

	buf, _ := utils.ReadFile("./params.yaml")

	actual := Params{}

	err := yaml.Unmarshal(buf, &actual)
	if err != nil {
		t.Errorf("Unable to decode into struct, %v", err)
	}

	if !reflect.DeepEqual(ExpectedGateway, actual.Gateway) {
		t.Errorf("Node object did not match expected value")
	}

}
