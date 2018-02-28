package node

import (
	"testing"
)

func TestID(t *testing.T) {
	// Set the ID. Only possible in node package
	id = 24

	// Try to modify the underlying ID and fail
	idResult := ID()
	idResult = 55
	t.Logf("Changed idResult to %v", idResult)

	expected := uint64(24)
	actual := ID()
	if actual != expected {
		t.Errorf("Got incorrect node ID. Got: %v, expected: %v\n", actual, expected)
	} else {
		println("TestID(): Passed")
	}
}
