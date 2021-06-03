///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package conf

import (
	"gitlab.com/xx_network/primitives/id"
)

var nodeID = id.ID([33]byte{82, 253, 252, 7, 33, 130, 101, 79, 22, 63, 95, 15,
	154, 98, 29, 114, 149, 102, 199, 77, 16, 3, 124, 77, 123, 187, 4, 7, 209,
	226, 198, 73, 2})

var ExpectedNode = Node{
	Paths:            ExpectedPaths,
	Port:             80,
	PublicAddress:    "127.0.0.1:80",
	ListeningAddress: "0.0.0.0:80",
}

/*
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

}*/
