////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

import (
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/utils"
	"gopkg.in/yaml.v2"
	"reflect"
	"testing"
)

var nodeID = id.ID([33]byte{82, 253, 252, 7, 33, 130, 101, 79, 22, 63, 95, 15,
	154, 98, 29, 114, 149, 102, 199, 77, 16, 3, 124, 77, 123, 187, 4, 7, 209,
	226, 198, 73, 2})

var ExpectedNode = Node{
	Paths:   ExpectedPaths,
	Address: "127.0.0.1:80",
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
		t.Errorf("Params node value does not match expected value"+
			"\n\texpected: %#v\n\treceived: %#v", ExpectedNode, actual.Node)
	}

}
