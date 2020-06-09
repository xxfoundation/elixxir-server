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

var ExpectedPermissioning = Permissioning{
	Paths: Paths{
		Cert: "~/.elixxir/permissioning.crt",
		Key:  "",
		Log:  "",
	},
	Address: "127.0.0.1:80",
}

// This test checks that unmarshalling the params.yaml file
// has the expected Permissioning object.
func TestPermissioning_UnmarshallingFileEqualsExpected(t *testing.T) {

	buf, _ := utils.ReadFile("./params.yaml")

	actual := Params{}

	err := yaml.Unmarshal(buf, &actual)
	if err != nil {
		t.Errorf("Unable to decode into struct, %v", err)
	}

	if !reflect.DeepEqual(ExpectedPermissioning, actual.Permissioning) {
		t.Errorf("Permissioning object did not match expected value")
	}

}
