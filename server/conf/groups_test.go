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

// This test checks that unmarshalling the groups.yaml file
// is equal to the expected groups object.
func TestGroups_UnmarshallingFileEqualsExpected(t *testing.T) {

	prime := large.NewInt(int64(17))
	smallPrime := large.NewInt(int64(11))
	generator := large.NewInt(int64(4))

	cmix := cyclic.NewGroup(prime, generator, smallPrime)
	e2e := cyclic.NewGroup(prime, generator, smallPrime)

	expected := Groups{
		CMix: cmix,
		E2E:  e2e,
	}

	actual := Groups{}
	buf, _ := ioutil.ReadFile("./Groups.yaml")

	err := yaml.Unmarshal(buf, &actual)
	if err != nil {
		t.Errorf("Unable to decode into struct, %v", err)
	}

	if actual.E2E.GetFingerprint() != expected.E2E.GetFingerprint() {
		t.Errorf("Groups object did not match expected values for E2E")
	}
	if actual.CMix.GetFingerprint() != expected.CMix.GetFingerprint() {
		t.Errorf("Groups object did not match expected values for E2E")
	}

}
