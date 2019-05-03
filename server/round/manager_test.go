////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package round

import (
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/services"
	"os"
	"testing"
	"time"
)

var mgr *Manager

func TestMain(m *testing.M) {
	mgr = NewManager()
	os.Exit(m.Run())
}

func TestManager(t *testing.T) {
	roundID := id.Round(58)
	round := New(grp, roundID, nil, nil, 0, 1)
	// Getting a round that's not been added should return nil
	result, err := mgr.GetRound(roundID)
	if result != nil || err == nil {
		t.Error("Shouldn't have gotten that round from the manager")
	}
	mgr.AddRound(round)
	// Getting a round that's been added should return that round
	result, _ = mgr.GetRound(roundID)
	if result.GetID() != roundID && err != nil {
		t.Errorf("Got round id %v from resulting round, expected %v",
			result.GetID(), roundID)
	}
	mgr.DeleteRound(roundID)
	// Getting a round that's been deleted should return nil
	result, err = mgr.GetRound(roundID)
	if result != nil || err == nil {
		t.Error("Shouldn't have gotten that round from the manager")
	}
}

func TestManager_GetPhase(t *testing.T) {
	roundID := id.Round(42)

	// Test round w/ nil phases
	round := New(grp, roundID, nil, nil, 0, 1)
	mgr.AddRound(round)
	p, err := mgr.GetPhase(roundID, 1)
	if err == nil {
		t.Errorf("GetPhase returned non-nil phase: %v", p)
	}

	roundID = id.Round(43)
	p, err = mgr.GetPhase(roundID, 1)
	if err == nil {
		t.Errorf("GetPhase returned non-nil phase: %v", p)
	}

	// Smoke test

	// We have to make phases with fake graphs...
	phases := make([]*phase.Phase, int(phase.NUM_PHASES))
	for i := 0; i < len(phases); i++ {
		phases[i] = phase.New(initMockGraph(services.
			NewGraphGenerator(1, nil, 1, 1, 1)),
			phase.Type(uint32(i)), nil,
			time.Second)
	}
	round = New(grp, roundID, phases, nil, 0, 1)
	mgr.AddRound(round)

	p, err = mgr.GetPhase(roundID, 0)
	if err != nil {
		t.Errorf("GetPhase returned nil phase: %v", err)
	}

	ty := p.GetType()
	if ty != phase.PrecompGeneration {
		t.Errorf("Returned phase of wrong type: %d", ty)
	}
}
