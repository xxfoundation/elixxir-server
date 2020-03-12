////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package receivers

import (
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/testUtil"
	"testing"
	"time"
)

func TestReceiveFinishRealtime(t *testing.T) {
	// Smoke tests the management part of PostPrecompResult
	grp := initImplGroup()
	const numNodes = 5

	resourceMonitor := measure.ResourceMonitor{}
	resourceMonitor.Set(&measure.ResourceMetric{})

	def := server.Definition{
		CmixGroup:       grp,
		Topology:        connect.NewCircuit(buildMockNodeIDs(numNodes)),
		UserRegistry:    &globals.UserMap{},
		ResourceMonitor: &resourceMonitor,
	}
	def.ID = def.Topology.GetNodeAtIndex(0)

	instance, _ := server.CreateServerInstance(&def, NewImplementation, true)

	instance.InitFirstNode()
	topology := instance.GetTopology()

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

	rnd := round.New(grp, nil, roundID, []phase.Phase{p}, responseMap, topology,
		topology.GetNodeAtIndex(0), 3, instance.GetRngStreamGen(),
		"0.0.0.0")

	instance.GetRoundManager().AddRound(rnd)

	// Initially, there should be zero rounds on the precomp queue
	if len(instance.GetFinishedRounds(t)) != 0 {
		t.Error("Expected completed precomps to be empty")
	}

	var err error

	info := mixmessages.RoundInfo{
		ID: uint64(roundID),
	}

	// Create a fake host and auth object to pass into function that needs it
	fakeHost, err := connect.NewHost(
		instance.GetTopology().GetLastNode().String(),
		"", nil, true, true)
	if err != nil {
		t.Errorf("Failed to create fakeHost, %s", err)
	}
	auth := connect.Auth{
		IsAuthenticated: true,
		Sender:          fakeHost,
	}

	go func() {
		err = ReceiveFinishRealtime(instance, &info, &auth)
	}()

	var finishedRoundID id.Round

	select {
	case finishedRoundID = <-instance.GetFinishedRounds(t):
	case <-time.After(2 * time.Second):
	}

	if err != nil {
		t.Errorf("ReceiveFinishRealtime: errored: %+v", err)
	}

	if finishedRoundID != roundID {
		t.Errorf("ReceiveFinishRealtime: Expected round %v to finish, "+
			"recieved %v", roundID, finishedRoundID)
	}
}

// Tests that the ReceiveFinishRealtime function will fail when passed with an
// auth object that has IsAuthenticated as false
func TestReceiveFinishRealtime_NoAuth(t *testing.T) {
	// Smoke tests the management part of PostPrecompResult
	grp := initImplGroup()
	const numNodes = 5

	resourceMonitor := measure.ResourceMonitor{}
	resourceMonitor.Set(&measure.ResourceMetric{})

	def := server.Definition{
		CmixGroup:       grp,
		Topology:        connect.NewCircuit(buildMockNodeIDs(numNodes)),
		UserRegistry:    &globals.UserMap{},
		ResourceMonitor: &resourceMonitor,
	}
	def.ID = def.Topology.GetNodeAtIndex(0)

	instance, _ := server.CreateServerInstance(&def, NewImplementation, false)

	instance.InitFirstNode()
	topology := instance.GetTopology()

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

	rnd := round.New(grp, nil, roundID, []phase.Phase{p}, responseMap, topology,
		topology.GetNodeAtIndex(0), 3, instance.GetRngStreamGen(),
		"0.0.0.0")

	instance.GetRoundManager().AddRound(rnd)

	// Initially, there should be zero rounds on the precomp queue
	if len(instance.GetFinishedRounds(t)) != 0 {
		t.Error("Expected completed precomps to be empty")
	}

	var err error

	info := mixmessages.RoundInfo{
		ID: uint64(roundID),
	}

	// Create a fake host and auth object to pass into function that needs it
	fakeHost, err := connect.NewHost(instance.GetTopology().GetNodeAtIndex(0).String(), "", nil, true, true)
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
	grp := initImplGroup()
	const numNodes = 5

	resourceMonitor := measure.ResourceMonitor{}
	resourceMonitor.Set(&measure.ResourceMetric{})

	def := server.Definition{
		CmixGroup:       grp,
		Topology:        connect.NewCircuit(buildMockNodeIDs(numNodes)),
		UserRegistry:    &globals.UserMap{},
		ResourceMonitor: &resourceMonitor,
	}
	def.ID = def.Topology.GetNodeAtIndex(0)

	instance, _ := server.CreateServerInstance(&def, NewImplementation, false)

	instance.InitFirstNode()
	topology := instance.GetTopology()

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

	rnd := round.New(grp, nil, roundID, []phase.Phase{p}, responseMap, topology,
		topology.GetNodeAtIndex(0), 3, instance.GetRngStreamGen(),
		"0.0.0.0")

	instance.GetRoundManager().AddRound(rnd)

	// Initially, there should be zero rounds on the precomp queue
	if len(instance.GetFinishedRounds(t)) != 0 {
		t.Error("Expected completed precomps to be empty")
	}

	var err error

	info := mixmessages.RoundInfo{
		ID: uint64(roundID),
	}

	// Create a fake host and auth object to pass into function that needs it
	fakeHost, err := connect.NewHost("bad", "", nil, true, true)
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
	grp := initImplGroup()
	const numNodes = 5

	resourceMonitor := measure.ResourceMonitor{}
	resourceMonitor.Set(&measure.ResourceMetric{})

	def := server.Definition{
		CmixGroup:       grp,
		Topology:        connect.NewCircuit(buildMockNodeIDs(numNodes)),
		UserRegistry:    &globals.UserMap{},
		ResourceMonitor: &resourceMonitor,
	}
	def.ID = def.Topology.GetNodeAtIndex(0)

	instance, _ := server.CreateServerInstance(&def, NewImplementation, false)

	instance.InitFirstNode()
	topology := instance.GetTopology()

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

	rnd := round.New(grp, nil, roundID, []phase.Phase{p}, responseMap, topology,
		topology.GetNodeAtIndex(0), 3, instance.GetRngStreamGen(),
		"0.0.0.0")

	instance.GetRoundManager().AddRound(rnd)

	// Initially, there should be zero rounds on the precomp queue
	if len(instance.GetFinishedRounds(t)) != 0 {
		t.Error("Expected completed precomps to be empty")
	}

	var err error

	info := mixmessages.RoundInfo{
		ID: uint64(roundID),
	}

	// Create a fake host and auth object to pass into function that needs it
	fakeHost, err := connect.NewHost(instance.GetTopology().GetNodeAtIndex(
		instance.GetTopology().Len()-1).String(), "", nil,
		true, true)
	if err != nil {
		t.Errorf("Failed to create fakeHost, %s", err)
	}
	auth := connect.Auth{
		IsAuthenticated: true,
		Sender:          fakeHost,
	}

	go func() {
		err = ReceiveFinishRealtime(instance, &info, &auth)
	}()

	var finishedRoundID id.Round

	select {
	case finishedRoundID = <-instance.GetFinishedRounds(t):
	case <-time.After(2 * time.Second):
	}

	if err != nil {
		t.Errorf("ReceiveFinishRealtime: errored: %+v", err)
	}

	if finishedRoundID != roundID {
		t.Errorf("ReceiveFinishRealtime: Expected round %v to finish, "+
			"recieved %v", roundID, finishedRoundID)
	}
}
