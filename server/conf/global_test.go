////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"reflect"
	"testing"
)

var ExpectedGlobal = Global{
	Batch:   uint32(20),
	SkipReg: true,
	Verbose: true,
	Groups:  ExpectedGroups,
}

// This test checks that unmarshalling the params.yaml file
// has the expected Global object.
func TestGlobal_UnmarshallingFileEqualsExpected(t *testing.T) {

	buf, _ := ioutil.ReadFile("./params.yaml")

	actual := Params{}

	err := yaml.Unmarshal(buf, &actual)
	if err != nil {
		t.Errorf("Unable to decode into struct, %v", err)
	}

	if !reflect.DeepEqual(ExpectedGlobal, actual.Global) {
		t.Errorf("Global object did not match expected value")
	}

}
