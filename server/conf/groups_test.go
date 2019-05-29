////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"testing"
)

var prime = large.NewInt(int64(17))
var smallPrime = large.NewInt(int64(11))
var generator = large.NewInt(int64(4))

var cmix = cyclic.NewGroup(prime, generator, smallPrime)
var e2e = cyclic.NewGroup(prime, generator, smallPrime)

var ExpectedGroups = Groups{
	CMix: cmix,
	E2E:  e2e,
}

// This test checks that unmarshalling the groups.yaml file
// is equal to the expected groups object.
func TestGroups_UnmarshallingFileEqualsExpected(t *testing.T) {

	actual := Params{}
	buf, _ := ioutil.ReadFile("./params.yaml")

	err := yaml.Unmarshal(buf, &actual)
	if err != nil {
		t.Errorf("Unable to decode into struct, %v", err)
	}

	if actual.Groups.E2E.GetFingerprint() != ExpectedGroups.E2E.GetFingerprint() {
		t.Errorf("Groups object did not match expected values for E2E")
	}
	if actual.Groups.CMix.GetFingerprint() != ExpectedGroups.CMix.GetFingerprint() {
		t.Errorf("Groups object did not match expected values for CMIX")
	}

}
