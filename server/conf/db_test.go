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

// This test checks that unmarshalling the db.yaml file
// is equal to the expected DB object.
func TestDB_UnmarshallingFileEqualsExpected(t *testing.T) {

	expected := DB{
		Name: "name",
		Username: "username",
		Password: "password",
		Addresses: []string{
			"127.0.0.1:80",
			"127.0.0.1:80",
			"127.0.0.1:80",
		},
	}

	buf, _ := ioutil.ReadFile("./db.yaml")
	actual := DB{}

	err := yaml.Unmarshal(buf, &actual)
	if err != nil {
		t.Errorf("Unable to decode into struct, %v", err)
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("DB object did not match expected values")
	}

}
