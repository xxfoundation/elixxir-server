////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/internal/state"
	"gitlab.com/elixxir/server/testUtil"
	"strings"
	"testing"
)

func createMockInstance(t *testing.T, instIndex int, s current.Activity) (*internal.Instance, *connect.Circuit, *cyclic.Group) {
	grp := initImplGroup()

	topology := connect.NewCircuit(BuildMockNodeIDs(5, t))
	def := internal.Definition{
		UserRegistry:    &globals.UserMap{},
		ResourceMonitor: &measure.ResourceMonitor{},
		FullNDF:         testUtil.NDF,
		PartialNDF:      testUtil.NDF,
		Gateway: internal.GW{
			ID: &id.TempGateway,
		},
		MetricsHandler: func(i *internal.Instance, roundID id.Round) error {
			return nil
		},
	}
	def.ID = topology.GetNodeAtIndex(instIndex)

	m := state.NewTestMachine(dummyStates, s, t)

	instance, _ := internal.CreateServerInstance(&def, NewImplementation, m, false)
	rnd, err := round.New(grp, nil, id.Round(0), make([]phase.Phase, 0),
		make(phase.ResponseMap), topology, topology.GetNodeAtIndex(0),
		3, instance.GetRngStreamGen(), nil, "0.0.0.0")
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}
	instance.GetRoundManager().AddRound(rnd)

	return instance, topology, grp
}

// Test caller function for PostRoundPublicKey
func TestPostRoundPublicKeyFunc(t *testing.T) {
	instance, topology, grp := createMockInstance(t, 1, current.PRECOMPUTING)

	batchSize := uint32(11)
	roundID := id.Round(0)

	mockPhase := testUtil.InitMockPhase(t)
	mockPhase.Ptype = phase.PrecompShare

	tagKey := mockPhase.GetType().String() + "Verification"
	responseMap := make(phase.ResponseMap)
	responseMap[tagKey] = phase.NewResponse(
		phase.ResponseDefinition{
			PhaseAtSource:  mockPhase.GetType(),
			ExpectedStates: []phase.State{phase.Active},
			PhaseToExecute: mockPhase.GetType()},
	)

	// Skip first node
	r, err := round.New(grp, instance.GetUserRegistry(), roundID,
		[]phase.Phase{mockPhase}, responseMap, topology,
		topology.GetNodeAtIndex(1), batchSize,
		instance.GetRngStreamGen(), nil, "0.0.0.0")
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}

	instance.GetRoundManager().AddRound(r)

	// Build a mock public key
	mockRoundInfo := &mixmessages.RoundInfo{ID: uint64(roundID)}
	mockPk := &mixmessages.RoundPublicKey{
		Round: mockRoundInfo,
		Key:   []byte{42},
	}

	impl := NewImplementation(instance)

	actualBatch := &mixmessages.Batch{}
	emptyBatch := &mixmessages.Batch{}
	impl.Functions.PostPhase = func(message *mixmessages.Batch, auth *connect.Auth) error {
		actualBatch = message
		return nil
	}

	fakeHost, err := connect.NewHost(topology.GetLastNode(), "", nil, true, true)
	if err != nil {
		t.Errorf("Failed to create fakeHost, %s", err)
	}
	auth := connect.Auth{
		IsAuthenticated: true,
		Sender:          fakeHost,
	}

	err = impl.Functions.PostRoundPublicKey(mockPk, &auth)
	if err != nil {
		t.Errorf("Failed to post round publickey: %+v", err)
	}

	// Verify that a PostPhase isn't called by ensuring callback
	// doesn't set the actual by comparing it to the empty batch
	if !batchEq(actualBatch, emptyBatch) {
		t.Errorf("Actual batch was not equal to empty batch in mock postphase")
	}

	if r.GetBuffer().CypherPublicKey.Cmp(grp.NewInt(42)) != 0 {
		// Error here
		t.Errorf("CypherPublicKey doesn't match expected value of the public key")
	}

}

// Test no auth error on ReceivePostRoundPublicKey
func TestReceivePostRoundPublicKey_AuthError(t *testing.T) {
	instance, topology, _ := createMockInstance(t, 1, current.PRECOMPUTING)

	fakeHost, _ := connect.NewHost(topology.GetLastNode(), "", nil, true, true)
	auth := &connect.Auth{
		IsAuthenticated: false,
		Sender:          fakeHost,
	}

	pk := &mixmessages.RoundPublicKey{
		Round: &mixmessages.RoundInfo{
			ID: 0,
		},
		Key: nil,
	}

	err := ReceivePostRoundPublicKey(instance, pk, auth)
	if err == nil {
		t.Error("ReceivePostRoundPublicKey did not return error when expected")
		return
	}

	if !strings.Contains(err.Error(), "Failed to authenticate") {
		t.Error("Did not receive expected authentication error")
	}
}

// Test bad host error on ReceivePostRoundPublicKey
func TestReceivePostRoundPublicKey_BadHostError(t *testing.T) {
	instance, _, _ := createMockInstance(t, 1, current.PRECOMPUTING)

	newID := id.NewIdFromString("beep beep i'm a host", id.Node, t)
	fakeHost, _ := connect.NewHost(newID, "", nil, true, true)
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          fakeHost,
	}

	pk := &mixmessages.RoundPublicKey{
		Round: &mixmessages.RoundInfo{
			ID: 0,
		},
		Key: nil,
	}

	err := ReceivePostRoundPublicKey(instance, pk, auth)
	if err == nil {
		t.Error("ReceivePostRoundPublicKey did not return error when expected")
		return
	}

	if !strings.Contains(err.Error(), "Failed to authenticate") {
		t.Error("Did not receive expected authentication error")
	}
}

// Test case in which PostRoundPublicKey is sent by first node
func TestPostRoundPublicKeyFunc_FirstNodeSendsBatch(t *testing.T) {
	instance, topology, grp := createMockInstance(t, 0, current.PRECOMPUTING)

	batchSize := uint32(3)
	roundID := id.Round(0)

	responseMap := make(phase.ResponseMap)

	mockPhaseShare := testUtil.InitMockPhase(t)
	mockPhaseShare.Ptype = phase.PrecompShare

	tagKey := mockPhaseShare.GetType().String() + "Verification"
	responseMap[tagKey] = phase.NewResponse(
		phase.ResponseDefinition{
			PhaseAtSource:  mockPhaseShare.GetType(),
			ExpectedStates: []phase.State{phase.Active},
			PhaseToExecute: mockPhaseShare.GetType()},
	)

	mockPhaseDecrypt := testUtil.InitMockPhase(t)
	mockPhaseDecrypt.Ptype = phase.PrecompDecrypt

	tagKey = mockPhaseDecrypt.GetType().String()
	responseMap[tagKey] = phase.NewResponse(
		phase.ResponseDefinition{
			PhaseAtSource:  mockPhaseDecrypt.GetType(),
			ExpectedStates: []phase.State{phase.Active},
			PhaseToExecute: mockPhaseDecrypt.GetType()},
	)

	// Don't skip first node
	r, err := round.New(grp, instance.GetUserRegistry(), roundID,
		[]phase.Phase{mockPhaseShare, mockPhaseDecrypt}, responseMap, topology,
		topology.GetNodeAtIndex(0), batchSize, instance.GetRngStreamGen(),
		nil, "0.0.0.0")
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}

	instance.GetRoundManager().AddRound(r)

	// Build a mock public key
	mockRoundInfo := &mixmessages.RoundInfo{ID: uint64(roundID)}
	mockPk := &mixmessages.RoundPublicKey{
		Round: mockRoundInfo,
		Key:   []byte{42},
	}

	impl := NewImplementation(instance)

	fakeHost, err := connect.NewHost(topology.GetLastNode(), "", nil, true, true)
	if err != nil {
		t.Errorf("Failed to create fakeHost, %s", err)
	}
	a := &connect.Auth{
		IsAuthenticated: true,
		Sender:          fakeHost,
	}
	err = impl.Functions.PostRoundPublicKey(mockPk, a)
	if err != nil {
		t.Errorf("Failed to PostRoundPublicKey: %+v", err)
	}

	// Verify that a PostPhase is called by ensuring callback
	// does set the actual by comparing it to the expected batch
	if uint32(len(mockPhaseDecrypt.GetIndices())) != batchSize {
		t.Errorf("first node did not recieve the correct number of " +
			"elements")
	}

	if r.GetBuffer().CypherPublicKey.Cmp(grp.NewInt(42)) != 0 {
		// Error here
		t.Errorf("CypherPublicKey doesn't match expected value of the " +
			"public key")
	}
}
