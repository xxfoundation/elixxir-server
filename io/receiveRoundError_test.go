///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package io

import (
	"crypto/rand"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/internal/state"
	"gitlab.com/elixxir/server/testUtil"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/signature"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"strings"
	"testing"
	"time"
)

// Happy path: Test if a properly crafted error message results in an error state
func TestReceiveRoundError(t *testing.T) {
	instance, topology, grp := setup_rounderror(t, 1, current.PRECOMPUTING)

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

	expectedError := "test failed"

	// Create a fake host and auth object to pass into function that needs it
	params := connect.GetDefaultHostParams()
	params.MaxRetries = 0
	fakeHost, err := connect.NewHost(topology.GetLastNode(), "", []byte(testUtil.RegCert), params)
	if err != nil {
		t.Errorf("Failed to create fakeHost, %s", err)
	}

	errMsg := &mixmessages.RoundError{
		Id:     uint64(roundID),
		Error:  expectedError,
		NodeId: topology.GetLastNode().Marshal(),
	}

	pk, err := rsa.LoadPrivateKeyFromPem([]byte(testUtil.RegPrivKey))
	if err != nil {
		t.Errorf("couldn't load privKey: %+v", err)
	}

	err = signature.Sign(errMsg, pk)
	if err != nil {
		t.Errorf("couldn't sign error message: %+v", err)
	}

	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          fakeHost,
	}

	err = ReceiveRoundError(errMsg, auth, instance)
	if err != nil {
		t.Errorf("Received error in ReceiveRoundError: %v", err)
		t.Fail()
	}

	time.Sleep(200 * time.Millisecond)

	// Check if error is passed along to channel
	receivedError := instance.GetRoundError()

	if receivedError == nil || strings.Compare(receivedError.Error, expectedError) != 0 {
		t.Errorf("Received error did not match expected. Expected: %s\n\tReceived",
			expectedError)
		t.Fail()
	}

	// Check if state has properly transition
	if instance.GetStateMachine().Get() != current.ERROR {
		t.Errorf("Failed to update to error state after ReceiveRoundError. We are in state: %v",
			instance.GetStateMachine().Get())
		t.Fail()
	}

}

// Error path: Check that if passed a round error with a node not in topology
//  it returns an error and does not transition to the error state
func TestReceiveRoundError_Auth(t *testing.T) {
	instance, topology, grp := setup_rounderror(t, 1, current.PRECOMPUTING)

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

	expectedError := "test failed"

	// Pass in an unknown node id to the message
	unknownNode := id.NewIdFromBytes([]byte("unknown"), t)
	errMsg := &mixmessages.RoundError{
		Id:     uint64(roundID),
		Error:  expectedError,
		NodeId: unknownNode.Marshal(),
	}

	// Create a fake host and auth object to pass into function that needs it
	params := connect.GetDefaultHostParams()
	params.MaxRetries = 0
	fakeHost, err := connect.NewHost(topology.GetLastNode(), "", nil, params)
	if err != nil {
		t.Errorf("Failed to create fakeHost, %s", err)
	}
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          fakeHost,
	}

	err = ReceiveRoundError(errMsg, auth, instance)
	if err != nil && !connect.IsAuthError(err) {
		t.Errorf("Received error in ReceiveRoundError: %v", err)
		t.Fail()
	}

}

// Error path: Check that if passed a round error with an invalid node id, that it properly errors
func TestReceiveRoundError_BadNodeId(t *testing.T) {
	instance, topology, grp := setup_rounderror(t, 1, current.PRECOMPUTING)

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

	expectedError := "test failed"

	// Pass in an invalid node id to the message
	errMsg := &mixmessages.RoundError{
		Id:     uint64(roundID),
		Error:  expectedError,
		NodeId: id.NewIdFromString("unknown", id.Node, t).Marshal(),
	}

	// Create a fake host and auth object to pass into function that needs it
	params := connect.GetDefaultHostParams()
	params.MaxRetries = 0
	fakeHost, err := connect.NewHost(topology.GetLastNode(),
		"", nil, params)
	if err != nil {
		t.Errorf("Failed to create fakeHost, %s", err)
	}
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          fakeHost,
	}

	err = ReceiveRoundError(errMsg, auth, instance)
	if err != nil && !strings.ContainsAny("Received unrecognizable node id", err.Error()) {
		t.Errorf("Received error in ReceiveRoundError: %v", err)
		t.Fail()
	}

}

// Error path: Craft message with a round unknown to the node. ReceiveRoundError should error
func TestReceiveRoundError_BadRound(t *testing.T) {
	instance, topology, grp := setup_rounderror(t, 1, current.PRECOMPUTING)

	// Set up a round
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

	expectedError := "test failed"

	// Pass in an unknown round id to the message
	errMsg := &mixmessages.RoundError{
		Id:     uint64(1),
		Error:  expectedError,
		NodeId: id.NewIdFromString("unknown", id.Node, t).Marshal(),
	}

	// Create a fake host and auth object to pass into function that needs it
	params := connect.GetDefaultHostParams()
	params.MaxRetries = 0
	fakeHost, err := connect.NewHost(topology.GetLastNode(), "", nil, params)
	if err != nil {
		t.Errorf("Failed to create fakeHost, %s", err)
	}
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          fakeHost,
	}

	err = ReceiveRoundError(errMsg, auth, instance)
	if err != nil && !strings.ContainsAny("Failed to get round 1", err.Error()) {
		t.Errorf("Received error in ReceiveRoundError: %v", err)
		t.Fail()
	}
}

func setup_rounderror(t *testing.T, instIndex int, s current.Activity) (*internal.Instance, *connect.Circuit, *cyclic.Group) {
	grp := initImplGroup()

	topology := connect.NewCircuit(BuildMockNodeIDs(5, t))
	def := internal.Definition{
		ResourceMonitor: &measure.ResourceMonitor{},
		FullNDF:         testUtil.NDF,
		PartialNDF:      testUtil.NDF,
		Flags:           internal.Flags{OverrideInternalIP: "0.0.0.0"},
		Gateway: internal.GW{
			ID: &id.TempGateway,
		},
		MetricsHandler: func(i *internal.Instance, roundID id.Round) error {
			return nil
		},
	}
	def.ID = topology.GetNodeAtIndex(instIndex)
	def.PrivateKey, _ = rsa.GenerateKey(rand.Reader, 1024)

	m := state.NewTestMachine(dummyStates, s, t)

	instance, _ := internal.CreateServerInstance(&def, NewImplementation, m, "1.1.0")
	rnd, err := round.New(grp, nil, id.Round(0), make([]phase.Phase, 0),
		make(phase.ResponseMap), topology, topology.GetNodeAtIndex(0),
		3, instance.GetRngStreamGen(), nil, "0.0.0.0", nil, nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}
	instance.GetRoundManager().AddRound(rnd)

	return instance, topology, grp
}
