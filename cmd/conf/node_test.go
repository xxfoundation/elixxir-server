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

var ExpectedNode = Node{
	Id: "Fb9JgRlv4AeF6EzgDNITvtK4dQRLc29nh3XtsLF86PE=",
	Ids: []string{"Fb9JgRlv4AeF6EzgDNITvtK4dQRLc29nh3XtsLF86PE=",
		"Fb9JgRlv4AeF6EzgDNITvtK4dQRLc29nh3XtsLF86PE=",
		"Fb9JgRlv4AeF6EzgDNITvtK4dQRLc29nh3XtsLF86PE="},
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

	buf, _ := utils.ReadFile("./params.yaml")

	actual := Params{}

	err := yaml.Unmarshal(buf, &actual)
	if err != nil {
		t.Errorf("Unable to decode into struct, %v", err)
	}

	if !reflect.DeepEqual(ExpectedNode, actual.Node) {
		t.Errorf("Params node value does not match expected value\nActual: %v"+
			"\nExpected: %v", actual.Node, ExpectedNode)
	}

}
