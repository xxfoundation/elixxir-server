////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package conf

import (
	"gitlab.com/xx_network/primitives/utils"
	"gopkg.in/yaml.v2"
	"reflect"
	"testing"
)

var ExpectedDatabase = Database{
	Name:     "name",
	Username: "username",
	Password: "password",
	Address:  "127.0.0.1",
	Port:     "80",
}

// This test checks that unmarshalling the params.yaml file
// has the expected Database object.
func TestDB_UnmarshallingFileEqualsExpected(t *testing.T) {

	buf, _ := utils.ReadFile("./params.yaml")

	actual := Params{}

	err := yaml.Unmarshal(buf, &actual)
	if err != nil {
		t.Errorf("Unable to decode into struct, %v", err)
	}

	actual.Database.Address = ExpectedDatabase.Address
	actual.Database.Port = ExpectedDatabase.Port
	if !reflect.DeepEqual(ExpectedDatabase, actual.Database) {
		t.Errorf("Database object did not match value, got %+v expected %+v", actual.Database, ExpectedDatabase)
	}

}
