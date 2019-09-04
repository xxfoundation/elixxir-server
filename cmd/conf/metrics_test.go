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

var ExpectedMetrics = Metrics{
	Log: "~/.elixxir/metrics.log",
}

// This test checks that unmarshalling the params.yaml file
// has the expected Database object.
func TestMetrics_UnmarshallingFileEqualsExpected(t *testing.T) {

	buf, _ := utils.ReadFile("./params.yaml")

	actual := Params{}

	err := yaml.Unmarshal(buf, &actual)
	if err != nil {
		t.Errorf("Unable to decode into struct, %v", err)
	}

	if !reflect.DeepEqual(ExpectedMetrics, actual.Metrics) {
		t.Errorf("Metrics object did not match value")
	}

}
