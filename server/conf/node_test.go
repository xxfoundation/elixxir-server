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

var ExpectedNode = Node{
	Id:    "pneumonoultramicroscopicsilicovolcanoconios=",
	Paths: ExpectedPaths,
	Addresses: []string{
		"127.0.0.1:80",
		"127.0.0.1:80",
		"127.0.0.1:80",
	},
}

// This test checks that unmarshalling the params.yaml file
// has the expected Node object.
func TestNode_UnmarshallingFileEqualsExpected(t *testing.T) {

	buf, _ := ioutil.ReadFile("./params.yaml")

	actual := Params{}

	err := yaml.Unmarshal(buf, &actual)
	if err != nil {
		t.Errorf("Unable to decode into struct, %v", err)
	}

	if !reflect.DeepEqual(ExpectedNode, actual.Node) {
		t.Errorf("Node object did not match expected value")
	}

}
