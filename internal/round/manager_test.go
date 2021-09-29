///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package round

import (
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/elixxir/server/storage"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"os"
	"runtime"
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
	round, err := New(grp, &storage.Storage{}, roundID, nil, nil, connect.NewCircuit([]*id.ID{{}}), &id.ID{}, 1, fastRNG.NewStreamGenerator(10000, uint(runtime.NumCPU()),
		csprng.NewSystemRNG), nil, "0.0.0.0", nil, nil, nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}
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
	round, err := New(grp, &storage.Storage{}, roundID, nil, nil, connect.NewCircuit([]*id.ID{{}}), &id.ID{}, 1, fastRNG.NewStreamGenerator(10000, uint(runtime.NumCPU()),
		csprng.NewSystemRNG), nil, "0.0.0.0", nil, nil, nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}
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
	phases := make([]phase.Phase, int(phase.NUM_PHASES))
	for i := 0; i < len(phases); i++ {
		gc := services.NewGraphGenerator(1,
			1, 1, 1)

		definition := phase.Definition{
			Graph:               initMockGraph(gc),
			Type:                phase.Type(uint32(i)),
			TransmissionHandler: nil,
			Timeout:             time.Second,
			DoVerification:      false,
		}

		phases[i] = phase.New(definition)
	}
	round, err = New(grp, &storage.Storage{}, roundID, phases, nil, connect.NewCircuit([]*id.ID{{}}), &id.ID{}, 1, fastRNG.NewStreamGenerator(10000, uint(runtime.NumCPU()),
		csprng.NewSystemRNG), nil, "0.0.0.0", nil, nil, nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}
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
