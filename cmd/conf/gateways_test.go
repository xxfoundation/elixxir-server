////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package conf

import (
	"gitlab.com/xx_network/primitives/utils"
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
