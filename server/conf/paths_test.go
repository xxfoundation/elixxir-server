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

// This test checks that unmarshalling the path.yaml file
// is equal to the expected Paths object.
func TestPaths_UnmarshallingFileEqualsExpected(t *testing.T) {

	expected := Paths{
		Cert: "~/.elixxir/cert.crt",
		Key:  "~/.elixxir/key.pem",
		Log:  "~/.elixxir/server.log",
	}

	buf, _ := ioutil.ReadFile("./paths.yaml")
	actual := Paths{}

	err := yaml.Unmarshal(buf, &actual)
	if err != nil {
		t.Errorf("Unable to decode into struct, %v", err)
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Paths object did not match expected values")
	}

}
