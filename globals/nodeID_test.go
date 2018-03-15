package globals

import (
	"testing"
	"github.com/spf13/viper"
	"math"
)

func nodeIDTestError(t *testing.T, actual, expected uint64) {
	if actual != expected {
		t.Errorf("NodeID: actual (%v) differed from expected (%v)", actual,
			expected)
	}
}

func TestNodeID(t *testing.T) {
	// first test: setting through ServerIdx if the viper variable isn't set
	actual := NodeID(489)
	expected := uint64(489)

	nodeIDTestError(t, actual, expected)

	// second test: setting through viper (this is done in cmd/root.go)
	nodeId = uint64(math.MaxUint64)
	viper.Set("nodeID", uint64(55))

	actual = NodeID(0)
	expected = uint64(55)

	nodeIDTestError(t, actual, expected)

	// third test: setting through viper to a number that uses all 64 bits
	nodeId = uint64(math.MaxUint64)
	viper.Set("nodeID", uint64(math.MaxUint64-5))

	actual = NodeID(0)
	expected = uint64(math.MaxUint64 - 5)
	nodeIDTestError(t, actual, expected)


	// fourth test: make sure you can't set the nodeId more than once
	actual = NodeID(67896)
	nodeIDTestError(t, actual, expected)
}
