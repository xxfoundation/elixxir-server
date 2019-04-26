package round

import (
	"gitlab.com/elixxir/primitives/id"
	"testing"
)

func TestManager(t *testing.T) {
	m := NewManager()
	roundID := id.Round(58)
	round := New(grp, roundID, nil, nil, 0, 1)
	// Getting a round that's not been added should return nil
	result := m.GetRound(roundID)
	if result != nil {
		t.Error("Shouldn't have gotten that round from the manager")
	}
	m.AddRound(round)
	// Getting a round that's been added should return that round
	result = m.GetRound(roundID)
	if result.GetID() != roundID {
		t.Errorf("Got round id %v from resulting round, expected %v",
			result.GetID(), roundID)
	}
	m.DeleteRound(roundID)
	// Getting a round that's been deleted should return nil
	result = m.GetRound(roundID)
	if result != nil {
		t.Error("Shouldn't have gotten that round from the manager")
	}
}
