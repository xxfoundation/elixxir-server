///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////
package io

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/comms/testkeys"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/testUtil"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/signature"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/utils"
	"testing"
)

func TestStartSharePhase(t *testing.T) {
	roundID := id.Round(0)
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2))

	instance, nodeAddr := mockInstance(t, mockSharePhaseImpl)
	topology := connect.NewCircuit([]*id.ID{instance.GetID()})

	cert, _ := utils.ReadFile(testkeys.GetNodeCertPath())
	nodeHost, _ := connect.NewHost(instance.GetID(), nodeAddr, cert, connect.GetDefaultHostParams())
	topology.AddHost(nodeHost)
	_, err := instance.GetNetwork().AddHost(instance.GetID(), nodeAddr, cert, connect.GetDefaultHostParams())
	if err != nil {
		t.Errorf("Failed to add host to instance: %v", err)
	}

	mockPhase := testUtil.InitMockPhase(t)
	responseMap := make(phase.ResponseMap)
	responseMap[phase.PrecompShare.String()] =
		phase.NewResponse(phase.ResponseDefinition{mockPhase.GetType(),
			[]phase.State{phase.Active}, mockPhase.GetType()})

	rnd, err := round.New(grp, &globals.UserMap{}, roundID, []phase.Phase{mockPhase},
		responseMap, topology, topology.GetNodeAtIndex(0), 3,
		instance.GetRngStreamGen(), nil, "0.0.0.0", nil, nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}
	instance.GetRoundManager().AddRound(rnd)

	ri := &mixmessages.RoundInfo{ID: uint64(roundID)}
	params := connect.GetDefaultHostParams()
	params.MaxRetries = 0

	testHost := topology.GetHostAtIndex(0)
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          testHost,
	}

	err = signature.Sign(ri, instance.GetPrivKey())
	if err != nil {
		t.Errorf("couldn't sign info message: %+v", err)
	}

	err = StartSharePhase(ri, auth, instance)
	if err != nil {
		t.Errorf("Error in happy path: %v", err)
	}
}

// Happy path (no final key logic in this test)
func TestSharePhaseRound(t *testing.T) {
	roundID := id.Round(0)
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2))

	instance, nodeAddr := mockInstance(t, mockSharePhaseImpl)
	// Fill with extra ID to avoid final Key generation codepath for this test
	mockId := id.NewIdFromBytes([]byte("test"), t)
	topology := connect.NewCircuit([]*id.ID{instance.GetID(), mockId})

	cert, _ := utils.ReadFile(testkeys.GetNodeCertPath())
	nodeHost, _ := connect.NewHost(instance.GetID(), nodeAddr, cert, connect.GetDefaultHostParams())
	topology.AddHost(nodeHost)
	_, err := instance.GetNetwork().AddHost(instance.GetID(), nodeAddr, cert, connect.GetDefaultHostParams())
	if err != nil {
		t.Errorf("Failed to add host to instance: %v", err)
	}

	mockPhase := testUtil.InitMockPhase(t)
	responseMap := make(phase.ResponseMap)
	responseMap[phase.PrecompShare.String()] =
		phase.NewResponse(phase.ResponseDefinition{mockPhase.GetType(),
			[]phase.State{phase.Active}, mockPhase.GetType()})

	rnd, err := round.New(grp, &globals.UserMap{}, roundID, []phase.Phase{mockPhase},
		responseMap, topology, topology.GetNodeAtIndex(0), 3,
		instance.GetRngStreamGen(), nil, "0.0.0.0", nil, nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}
	instance.GetRoundManager().AddRound(rnd)

	ri := &mixmessages.RoundInfo{ID: uint64(roundID)}
	params := connect.GetDefaultHostParams()
	params.MaxRetries = 0

	testHost := topology.GetHostAtIndex(0)
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          testHost,
	}

	err = signature.Sign(ri, instance.GetPrivKey())
	if err != nil {
		t.Errorf("couldn't sign info message: %+v", err)
	}

	piece := generateShare(nil, grp, rnd, instance.GetID())

	err = signature.Sign(piece, instance.GetPrivKey())
	if err != nil {
		t.Errorf("couldn't sign info message: %+v", err)
	}

	err = SharePhaseRound(piece, auth, instance)
	if err != nil {
		t.Errorf("Error in happy path: %v", err)
	}
}

func TestSharePhaseRound_FinalKey(t *testing.T) {
	roundID := id.Round(0)
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2))

	instance, nodeAddr := mockInstance(t, mockSharePhaseImpl)
	topology := connect.NewCircuit([]*id.ID{instance.GetID()})

	cert, _ := utils.ReadFile(testkeys.GetNodeCertPath())
	nodeHost, _ := connect.NewHost(instance.GetID(), nodeAddr, cert, connect.GetDefaultHostParams())
	topology.AddHost(nodeHost)
	_, err := instance.GetNetwork().AddHost(instance.GetID(), nodeAddr, cert, connect.GetDefaultHostParams())
	if err != nil {
		t.Errorf("Failed to add host to instance: %v", err)
	}

	mockPhase := testUtil.InitMockPhase(t)
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

	responseMap[phase.PrecompShare.String()] =
		phase.NewResponse(phase.ResponseDefinition{mockPhase.GetType(),
			[]phase.State{phase.Active}, mockPhase.GetType()})
	responseMap[phase.PrecompShare.String()+"Verification"] =
		phase.NewResponse(phase.ResponseDefinition{mockPhase.GetType(),
			[]phase.State{phase.Active}, mockPhase.GetType()})

	rnd, err := round.New(grp, &globals.UserMap{}, roundID, []phase.Phase{mockPhase},
		responseMap, topology, topology.GetNodeAtIndex(0), 3,
		instance.GetRngStreamGen(), nil, "0.0.0.0", nil, nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}
	instance.GetRoundManager().AddRound(rnd)

	ri := &mixmessages.RoundInfo{ID: uint64(roundID)}
	params := connect.GetDefaultHostParams()
	params.MaxRetries = 0

	testHost := topology.GetHostAtIndex(0)
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          testHost,
	}

	err = signature.Sign(ri, instance.GetPrivKey())
	if err != nil {
		t.Errorf("couldn't sign info message: %+v", err)
	}

	piece := generateShare(nil, grp, rnd, instance.GetID())

	err = signature.Sign(piece, instance.GetPrivKey())
	if err != nil {
		t.Errorf("couldn't sign info message: %+v", err)
	}

	err = SharePhaseRound(piece, auth, instance)
	if err != nil {
		t.Errorf("Error in happy path: %v", err)
	}

	// Check that the key has been modified in the round
	expectedKey := grp.NewIntFromBytes(piece.Piece)
	receivedKey := rnd.GetBuffer().CypherPublicKey
	if expectedKey.Cmp(receivedKey) != 0 {
		t.Errorf("Final key did not match expected."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", expectedKey.Bytes(), receivedKey.Bytes())
	}
}

func mockSharePhaseImpl(instance *internal.Instance) *node.Implementation {
	impl := node.NewImplementation()
	impl.Functions.SharePhaseRound = func(sharedPiece *mixmessages.SharePiece,
		auth *connect.Auth) error {
		return nil
	}
	impl.Functions.StartSharePhase = func(ri *mixmessages.RoundInfo, auth *connect.Auth) error {
		return nil
	}

	return impl
}
