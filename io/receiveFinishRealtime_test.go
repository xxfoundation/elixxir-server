///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/testUtil"
	"gitlab.com/xx_network/comms/connect"
	"testing"
)

func TestReceiveFinishRealtime(t *testing.T) {
	instance, topology, grp := createMockInstance(t, 0, current.REALTIME)

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
		"0.0.0.0", nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}

	instance.GetRoundManager().AddRound(rnd)

	info := mixmessages.RoundInfo{
		ID: uint64(roundID),
	}

	// Create a fake host and auth object to pass into function that needs it
	fakeHost, err := connect.NewHost(topology.GetLastNode(), "", nil, true, true)
	if err != nil {
		t.Errorf("Failed to create fakeHost, %s", err)
	}
	auth := connect.Auth{
		IsAuthenticated: true,
		Sender:          fakeHost,
	}

	err = ReceiveFinishRealtime(instance, &info, &auth)

	if err != nil {
		t.Errorf("ReceiveFinishRealtime: errored: %+v", err)
	}

}

// Tests that the ReceiveFinishRealtime function will fail when passed with an
// auth object that has IsAuthenticated as false
func TestReceiveFinishRealtime_NoAuth(t *testing.T) {
	// Smoke tests the management part of PostPrecompResult
	resourceMonitor := measure.ResourceMonitor{}
	resourceMonitor.Set(measure.ResourceMetric{})
	instance, topology, grp := createMockInstance(t, 0, current.REALTIME)

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
		"0.0.0.0", nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}

	instance.GetRoundManager().AddRound(rnd)

	info := mixmessages.RoundInfo{
		ID: uint64(roundID),
	}

	// Create a fake host and auth object to pass into function that needs it
	fakeHost, err := connect.NewHost(topology.GetNodeAtIndex(0), "", nil, true, true)
	if err != nil {
		t.Errorf("Failed to create fakeHost, %s", err)
	}
	auth := connect.Auth{
		IsAuthenticated: false,
		Sender:          fakeHost,
	}

	err = ReceiveFinishRealtime(instance, &info, &auth)
	if err == nil {
		t.Errorf("ReceiveFinishRealtime: did not error with IsAuthenticated false")
	}
}

// Tests that the ReceiveFinishRealtime function will fail when passed with an
// auth object that has Sender as something that isn't the right node for the
// call
func TestReceiveFinishRealtime_WrongSender(t *testing.T) {
	// Smoke tests the management part of PostPrecompResult
	const numNodes = 5
	resourceMonitor := measure.ResourceMonitor{}
	resourceMonitor.Set(measure.ResourceMetric{})

	instance, topology, grp := createMockInstance(t, 0, current.REALTIME)

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
		"0.0.0.0", nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}

	instance.GetRoundManager().AddRound(rnd)

	info := mixmessages.RoundInfo{
		ID: uint64(roundID),
	}

	// Create a fake host and auth object to pass into function that needs it
	newID := id.NewIdFromString("bad", id.Node, t)
	fakeHost, err := connect.NewHost(newID, "", nil, true, true)
	if err != nil {
		t.Errorf("Failed to create fakeHost, %s", err)
	}
	auth := connect.Auth{
		IsAuthenticated: true,
		Sender:          fakeHost,
	}

	err = ReceiveFinishRealtime(instance, &info, &auth)
	if err == nil {
		t.Errorf("ReceiveFinishRealtime: did not error with wrong host")
	}
}

func TestReceiveFinishRealtime_GetMeasureHandler(t *testing.T) {
	// Smoke tests the management part of PostPrecompResult
	const numNodes = 5

	resourceMonitor := measure.ResourceMonitor{}
	resourceMonitor.Set(measure.ResourceMetric{})

	instance, topology, grp := createMockInstance(t, 0, current.REALTIME)

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
		"0.0.0.0", nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}

	instance.GetRoundManager().AddRound(rnd)

	info := mixmessages.RoundInfo{
		ID: uint64(roundID),
	}

	// Create a fake host and auth object to pass into function that needs it
	fakeHost, err := connect.NewHost(topology.GetNodeAtIndex(topology.Len()-1),
		"", nil, true, true)
	if err != nil {
		t.Errorf("Failed to create fakeHost, %s", err)
	}
	auth := connect.Auth{
		IsAuthenticated: true,
		Sender:          fakeHost,
	}

	err = ReceiveFinishRealtime(instance, &info, &auth)

	if err != nil {
		t.Errorf("ReceiveFinishRealtime: errored: %+v", err)
	}

}
