///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package io

import (
	"encoding/json"
	"git.xx.network/elixxir/comms/mixmessages"
	"git.xx.network/elixxir/server/internal"
	"git.xx.network/elixxir/server/internal/measure"
	"git.xx.network/elixxir/server/internal/phase"
	"git.xx.network/elixxir/server/internal/round"
	"git.xx.network/elixxir/server/internal/state"
	"git.xx.network/elixxir/server/testUtil"
	"git.xx.network/xx_network/comms/connect"
	"git.xx.network/xx_network/primitives/id"
	"sync"
	"testing"
	"time"
)

func TestReceiveGetMeasure(t *testing.T) {
	// Smoke tests the management part of PostPrecompResult
	grp := initImplGroup()
	const numNodes = 5
	var err error

	resourceMonitor := measure.ResourceMonitor{}
	resourceMonitor.Set(measure.ResourceMetric{})
	topology := connect.NewCircuit(BuildMockNodeIDs(numNodes, t))
	// Set instance for first node

	m := state.NewMachine(dummyStates)

	metric := measure.ResourceMetric{
		SystemStartTime: time.Time{},
		Time:            time.Time{},
		MemAllocBytes:   0,
		MemAvailable:    0,
		NumThreads:      0,
		CPUPercentage:   0,
	}

	monitor := measure.ResourceMonitor{RWMutex: sync.RWMutex{}}
	monitor.Set(metric)
	//nid := server.GenerateId(t)
	def := internal.Definition{
		ID:              topology.GetNodeAtIndex(0),
		ResourceMonitor: &monitor,
		FullNDF:         testUtil.NDF,
		PartialNDF:      testUtil.NDF,
		Flags:           internal.Flags{OverrideInternalIP: "0.0.0.0"},
		DevMode:         true,
	}
	def.Gateway.ID = def.ID.DeepCopy()
	def.Gateway.ID.SetType(id.Gateway)

	instance, _ := internal.CreateServerInstance(&def, NewImplementation, m, "1.1.0")

	// Set up a round first node
	roundID := id.Round(45)

	response := phase.NewResponse(phase.ResponseDefinition{
		PhaseAtSource:  phase.RealPermute,
		ExpectedStates: []phase.State{phase.Active},
		PhaseToExecute: phase.RealPermute})

	responseMap := make(phase.ResponseMap)
	responseMap["RealPermuteVerification"] = response

	p := testUtil.InitMockPhase(t)
	p.Ptype = phase.RealPermute

	rnd, err := round.New(grp, nil, roundID, []phase.Phase{p}, responseMap, topology,
		topology.GetNodeAtIndex(0), 3, instance.GetRngStreamGen(), nil,
		"0.0.0.0", nil, nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}

	instance.GetRoundManager().AddRound(rnd)

	var resp *mixmessages.RoundMetrics

	info := mixmessages.RoundInfo{
		ID: uint64(roundID),
	}

	// Build a host around the last node
	firstNodeId := topology.GetNodeAtIndex(0)
	fakeHost, err := connect.NewHost(firstNodeId, "", nil, connect.GetDefaultHostParams())
	if err != nil {
		t.Errorf("Failed to create fakeHost, %s", err)
	}

	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          fakeHost,
	}

	rnd.GetMeasurementsReadyChan() <- struct{}{}

	resp, err = ReceiveGetMeasure(instance, &info, auth)

	if err != nil {
		t.Errorf("Failed to return metrics: %+v", err)
	}
	remade := *new(measure.RoundMetrics)

	err = json.Unmarshal([]byte(resp.RoundMetricJSON), &remade)

	if err != nil {
		t.Errorf("Failed to extract data from JSON: %+v", err)
	}

	info = mixmessages.RoundInfo{
		ID: uint64(roundID) - 1,
	}

	_, err = ReceiveGetMeasure(instance, &info, auth)

	if err == nil {
		t.Errorf("This should have thrown an error, instead got: %+v", err)
	}
}
