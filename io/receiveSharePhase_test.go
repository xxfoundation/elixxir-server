///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////
package io

import (
	"testing"
	"time"

	"gitlab.com/elixxir/comms/mixmessages"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/comms/testkeys"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/internal/state"
	"gitlab.com/elixxir/server/storage"
	"gitlab.com/elixxir/server/testUtil"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/signature"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/utils"
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

	rnd, err := round.New(grp, &storage.Storage{}, roundID, []phase.Phase{mockPhase}, responseMap, topology, topology.GetNodeAtIndex(0), 3, instance.GetRngStreamGen(), nil, "0.0.0.0", nil, nil, nil)
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

	err = signature.SignRsa(ri, instance.GetPrivKey())
	if err != nil {
		t.Errorf("couldn't sign info message: %+v", err)
	}

	err = ReceiveStartSharePhase(ri, auth, instance)
	if err != nil {
		t.Errorf("Error in happy path: %v", err)
	}
}

func TestReceiveStartSharePhase_BadAuth(t *testing.T) {
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

	rnd, err := round.New(grp, &storage.Storage{}, roundID, []phase.Phase{mockPhase}, responseMap, topology, topology.GetNodeAtIndex(0), 3, instance.GetRngStreamGen(), nil, "0.0.0.0", nil, nil, nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}
	instance.GetRoundManager().AddRound(rnd)

	ri := &mixmessages.RoundInfo{ID: uint64(roundID)}
	params := connect.GetDefaultHostParams()
	params.MaxRetries = 0

	testHost, err := connect.NewHost(id.NewIdFromBytes([]byte("badID"), t),
		"0.0.0.0", nil, connect.GetDefaultHostParams())
	if err != nil {
		t.Fatalf("Could not construct mock host: %v", err)
	}
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          testHost,
	}

	err = signature.SignRsa(ri, instance.GetPrivKey())
	if err != nil {
		t.Errorf("couldn't sign info message: %+v", err)
	}

	err = ReceiveStartSharePhase(ri, auth, instance)
	if err == nil {
		t.Errorf("Auth check should fail in error path")
	}

}

// Happy path (no final key logic in this test)
func TestSharePhaseRound(t *testing.T) {
	roundID := id.Round(0)
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2))

	// Build an instance and dummy host.
	// Dummy blocks final key logic from executing
	instance, nodeAddr := mockInstance(t, mockSharePhaseImpl)
	dummyNode, dummyAddr := mockInstance(t, dummySharePhaseImpl)

	// Fill with extra ID to avoid final Key generation codepath for this test
	topology := connect.NewCircuit([]*id.ID{instance.GetID(), dummyNode.GetID()})

	// Create host objects off of the nodes
	cert, _ := utils.ReadFile(testkeys.GetNodeCertPath())
	nodeHost, _ := connect.NewHost(instance.GetID(), nodeAddr, cert, connect.GetDefaultHostParams())
	mockHost, _ := connect.NewHost(dummyNode.GetID(), dummyAddr, cert, connect.GetDefaultHostParams())

	// Add both to the topology
	topology.AddHost(nodeHost)
	topology.AddHost(mockHost)

	// Add to instance itself and the dummy node as hosts
	_, err := instance.GetNetwork().AddHost(instance.GetID(), nodeAddr, cert, connect.GetDefaultHostParams())
	if err != nil {
		t.Errorf("Failed to add host to instance: %v", err)
	}
	_, err = instance.GetNetwork().AddHost(dummyNode.GetID(), dummyAddr, cert, connect.GetDefaultHostParams())
	if err != nil {
		t.Errorf("Failed to add host to instance: %v", err)
	}

	// Add to dummy itself and the instance as hosts
	_, err = dummyNode.GetNetwork().AddHost(instance.GetID(), nodeAddr, cert, connect.GetDefaultHostParams())
	if err != nil {
		t.Errorf("Failed to add host to dummy node: %v", err)
	}
	_, err = dummyNode.GetNetwork().AddHost(dummyNode.GetID(), nodeAddr, cert, connect.GetDefaultHostParams())
	if err != nil {
		t.Errorf("Failed to add host to dummy node: %v", err)
	}

	// Build phase handling
	mockPhase := testUtil.InitMockPhase(t)
	responseMap := make(phase.ResponseMap)
	responseMap[phase.PrecompShare.String()] =
		phase.NewResponse(phase.ResponseDefinition{mockPhase.GetType(),
			[]phase.State{phase.Active}, mockPhase.GetType()})

	// Build round and add it to the manager
	rnd, err := round.New(grp, &storage.Storage{}, roundID, []phase.Phase{mockPhase}, responseMap, topology, topology.GetNodeAtIndex(0), 3, instance.GetRngStreamGen(), nil, "0.0.0.0", nil, nil, nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}
	instance.GetRoundManager().AddRound(rnd)

	params := connect.GetDefaultHostParams()
	params.MaxRetries = 0

	// Get the previous node host for proper auth validation
	testHost := topology.GetHostAtIndex(1)
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          testHost,
	}

	instance.GetPhaseShareMachine().Update(state.STARTED)

	// Generate a share to send
	msg := &pb.SharePiece{
		Piece:        grp.GetG().Bytes(),
		Participants: make([][]byte, 0),
		RoundID:      uint64(rnd.GetID()),
	}
	piece, err := generateShare(msg, grp, rnd, instance)
	if err != nil {
		t.Errorf("Could not generate a mock share: %v", err)
	}
	// Manually fudge participant list so
	// our instance is not in the list
	piece.Participants = [][]byte{dummyNode.GetID().Bytes()}

	err = ReceiveSharePhasePiece(piece, auth, instance)
	if err != nil {
		t.Errorf("Error in happy path: %v", err)
	}
}

// Error path
func TestReceiveSharePhasePiece_BadAuth(t *testing.T) {
	roundID := id.Round(0)
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2))

	// Build an instance and dummy host.
	// Dummy blocks final key logic from executing
	instance, nodeAddr := mockInstance(t, mockSharePhaseImpl)
	dummyNode, dummyAddr := mockInstance(t, dummySharePhaseImpl)

	// Fill with extra ID to avoid final Key generation codepath for this test
	topology := connect.NewCircuit([]*id.ID{instance.GetID(), dummyNode.GetID()})

	// Create host objects off of the nodes
	cert, _ := utils.ReadFile(testkeys.GetNodeCertPath())
	nodeHost, _ := connect.NewHost(instance.GetID(), nodeAddr, cert, connect.GetDefaultHostParams())
	mockHost, _ := connect.NewHost(dummyNode.GetID(), dummyAddr, cert, connect.GetDefaultHostParams())

	// Add both to the topology
	topology.AddHost(nodeHost)
	topology.AddHost(mockHost)

	// Add to instance itself and the dummy node as hosts
	_, err := instance.GetNetwork().AddHost(instance.GetID(), nodeAddr, cert, connect.GetDefaultHostParams())
	if err != nil {
		t.Errorf("Failed to add host to instance: %v", err)
	}
	_, err = instance.GetNetwork().AddHost(dummyNode.GetID(), dummyAddr, cert, connect.GetDefaultHostParams())
	if err != nil {
		t.Errorf("Failed to add host to instance: %v", err)
	}

	// Add to dummy itself and the instance as hosts
	_, err = dummyNode.GetNetwork().AddHost(instance.GetID(), nodeAddr, cert, connect.GetDefaultHostParams())
	if err != nil {
		t.Errorf("Failed to add host to dummy node: %v", err)
	}
	_, err = dummyNode.GetNetwork().AddHost(dummyNode.GetID(), nodeAddr, cert, connect.GetDefaultHostParams())
	if err != nil {
		t.Errorf("Failed to add host to dummy node: %v", err)
	}

	// Build phase handling
	mockPhase := testUtil.InitMockPhase(t)
	responseMap := make(phase.ResponseMap)
	responseMap[phase.PrecompShare.String()] =
		phase.NewResponse(phase.ResponseDefinition{mockPhase.GetType(),
			[]phase.State{phase.Active}, mockPhase.GetType()})

	// Build round and add it to the manager
	rnd, err := round.New(grp, &storage.Storage{}, roundID, []phase.Phase{mockPhase}, responseMap, topology, topology.GetNodeAtIndex(0), 3, instance.GetRngStreamGen(), nil, "0.0.0.0", nil, nil, nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}
	instance.GetRoundManager().AddRound(rnd)

	params := connect.GetDefaultHostParams()
	params.MaxRetries = 0

	// Get the previous node host for proper auth validation
	testHost, err := connect.NewHost(id.NewIdFromBytes([]byte("badID"), t),
		"0.0.0.0", nil, connect.GetDefaultHostParams())
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          testHost,
	}

	instance.GetPhaseShareMachine().Update(state.STARTED)

	// Generate a share to send
	msg := &pb.SharePiece{
		Piece:        grp.GetG().Bytes(),
		Participants: make([][]byte, 0),
		RoundID:      uint64(rnd.GetID()),
	}
	piece, err := generateShare(msg, grp, rnd, instance)
	if err != nil {
		t.Errorf("Could not generate a mock share: %v", err)
	}
	// Manually fudge participant list so
	// our instance is not in the list
	piece.Participants = [][]byte{dummyNode.GetID().Bytes()}

	err = ReceiveSharePhasePiece(piece, auth, instance)
	if err == nil {
		t.Errorf("Auth check should fail in error path")
	}

}

// Test implicit call ReceiveFinalKey through
// a ReceiveSharePhasePiece call
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

	responseMap := make(phase.ResponseMap)
	mockPhaseShare := testUtil.InitMockPhase(t)
	mockPhaseShare.Ptype = phase.PrecompShare

	mockPhaseDecrypt := testUtil.InitMockPhase(t)
	mockPhaseDecrypt.Ptype = phase.PrecompDecrypt

	phases := []phase.Phase{mockPhaseShare, mockPhaseDecrypt}

	rnd, err := round.New(grp, &storage.Storage{}, roundID, phases, responseMap, topology, topology.GetNodeAtIndex(0), 3, instance.GetRngStreamGen(), nil, "0.0.0.0", nil, nil, nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}
	instance.GetRoundManager().AddRound(rnd)

	params := connect.GetDefaultHostParams()
	params.MaxRetries = 0

	testHost := topology.GetHostAtIndex(0)
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          testHost,
	}

	// Generate a mock message
	msg := &pb.SharePiece{
		Piece:        grp.GetG().Bytes(),
		Participants: make([][]byte, 0),
		RoundID:      uint64(rnd.GetID()),
	}
	piece, err := generateShare(msg, grp, rnd, instance)
	if err != nil {
		t.Errorf("Could not generate a mock share: %v", err)
	}

	ok, err := instance.GetPhaseShareMachine().Update(state.STARTED)
	if !ok || err != nil {
		t.Errorf("Trouble updating phase machine for test: %v", err)
	}

	err = ReceiveSharePhasePiece(piece, auth, instance)
	if err != nil {
		t.Errorf("Error in happy path: %v", err)
	}

	time.Sleep(5 * time.Second)

	// Check that the key has been modified in the round
	expectedKey := grp.NewIntFromBytes(piece.Piece)
	receivedKey := rnd.GetBuffer().CypherPublicKey
	if expectedKey.Cmp(receivedKey) != 0 {
		t.Errorf("Final key did not match expected."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", expectedKey.Bytes(), receivedKey.Bytes())
	}
}

// Unit test
func TestReceiveFinalKey(t *testing.T) {
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

	// Build responses for checks and for phase transition
	responseMap := make(phase.ResponseMap)
	mockPhaseShare := testUtil.InitMockPhase(t)
	mockPhaseShare.Ptype = phase.PrecompShare

	mockPhaseDecrypt := testUtil.InitMockPhase(t)
	mockPhaseDecrypt.Ptype = phase.PrecompDecrypt

	phases := []phase.Phase{mockPhaseShare, mockPhaseDecrypt}

	rnd, err := round.New(grp, &storage.Storage{}, roundID, phases, responseMap, topology, topology.GetNodeAtIndex(0), 3, instance.GetRngStreamGen(), nil, "0.0.0.0", nil, nil, nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}
	instance.GetRoundManager().AddRound(rnd)

	testHost := topology.GetHostAtIndex(0)
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          testHost,
	}

	// Generate a mock message
	msg := &pb.SharePiece{
		Piece:        grp.GetG().Bytes(),
		Participants: make([][]byte, 0),
		RoundID:      uint64(rnd.GetID()),
	}
	piece, err := generateShare(msg, grp, rnd, instance)
	if err != nil {
		t.Errorf("Could not generate a mock share: %v", err)
	}

	ok, err := instance.GetPhaseShareMachine().Update(state.STARTED)
	if !ok || err != nil {
		t.Errorf("Trouble updating phase machine for test: %v", err)
	}

	err = ReceiveFinalKey(piece, auth, instance)
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

// Error path
func TestReceiveFinalKey_BadAuth(t *testing.T) {
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

	// Build responses for checks and for phase transition
	responseMap := make(phase.ResponseMap)
	mockPhaseShare := testUtil.InitMockPhase(t)
	mockPhaseShare.Ptype = phase.PrecompShare

	mockPhaseDecrypt := testUtil.InitMockPhase(t)
	mockPhaseDecrypt.Ptype = phase.PrecompDecrypt

	phases := []phase.Phase{mockPhaseShare, mockPhaseDecrypt}

	rnd, err := round.New(grp, &storage.Storage{}, roundID, phases, responseMap, topology, topology.GetNodeAtIndex(0), 3, instance.GetRngStreamGen(), nil, "0.0.0.0", nil, nil, nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}
	instance.GetRoundManager().AddRound(rnd)

	testHost, err := connect.NewHost(id.NewIdFromBytes([]byte("badID"), t),
		"0.0.0.0", nil, connect.GetDefaultHostParams())
	if err != nil {
		t.Fatalf("Could not construct mock host: %v", err)
	}
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          testHost,
	}

	// Generate a mock message
	msg := &pb.SharePiece{
		Piece:        grp.GetG().Bytes(),
		Participants: make([][]byte, 0),
		RoundID:      uint64(rnd.GetID()),
	}
	piece, err := generateShare(msg, grp, rnd, instance)
	if err != nil {
		t.Errorf("Could not generate a mock share: %v", err)
	}

	ok, err := instance.GetPhaseShareMachine().Update(state.STARTED)
	if !ok || err != nil {
		t.Errorf("Trouble updating phase machine for test: %v", err)
	}

	err = ReceiveFinalKey(piece, auth, instance)
	if err == nil {
		t.Errorf("Should not validate auth check in error path")
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
	impl.Functions.ShareFinalKey = func(sharedPiece *mixmessages.SharePiece, auth *connect.Auth) error {
		return ReceiveFinalKey(sharedPiece, auth, instance)
	}

	return impl
}

func dummySharePhaseImpl(instance *internal.Instance) *node.Implementation {
	impl := node.NewImplementation()
	impl.Functions.SharePhaseRound = func(sharedPiece *mixmessages.SharePiece,
		auth *connect.Auth) error {
		return nil
	}
	impl.Functions.StartSharePhase = func(ri *mixmessages.RoundInfo, auth *connect.Auth) error {
		return nil
	}
	impl.Functions.ShareFinalKey = func(sharedPiece *mixmessages.SharePiece, auth *connect.Auth) error {
		return nil
	}

	return impl
}
