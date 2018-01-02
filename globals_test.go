package globals

import (
	"testing"
)

func TestNewRound(t *testing.T) {
	var actual, expected *round
	expected = nil
	actual = NewRound(42)
	if (actual != expected) {
		t.Errorf("Expected: %v, got: %v", expected, actual)
	}
}
