////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"testing"
)

// This test checks that unmarshalling the params.yaml file
// is equal to the expected Params object.
func TestParams_UnmarshallingFileEqualsExpected(t *testing.T) {

	buf, _ := ioutil.ReadFile("./params.yaml")
	actual := Params{}

	err := yaml.Unmarshal(buf, &actual)
	if err != nil {
		t.Errorf("Unable to decode into struct, %v", err)
	}

	if actual.NodeID != 100 {
		t.Errorf("Params object did not match expected value for NodeID")
	}

	if actual.SkipReg != true {
		t.Errorf("Params object did not match expected value for SkipReg")
	}

}
