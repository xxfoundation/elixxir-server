///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package io

import (
	"context"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/testUtil"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/primitives/id"
	"google.golang.org/grpc/metadata"
	"io"
	"strings"
	"testing"
)

// Smoke test
func TestReceivePrecompTestBatch(t *testing.T) {
	instance, topology, grp := createMockInstance(t, 1, current.PRECOMPUTING)

	// Build a host around the last node
	lastNodeIndex := topology.Len() - 1
	lastNodeId := topology.GetNodeAtIndex(lastNodeIndex)
	params := connect.GetDefaultHostParams()
	params.MaxRetries = 0
	fakeHost, err := connect.NewHost(lastNodeId, "", nil, params)
	if err != nil {
		t.Errorf("Failed to create fakeHost, %s", err)
	}

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

	batchSize := 3
	rnd, err := round.New(grp, nil, roundID, []phase.Phase{p}, responseMap, topology, topology.GetNodeAtIndex(0), uint32(batchSize), instance.GetRngStreamGen(), nil, "0.0.0.0", nil, nil, nil, nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}

	instance.GetRoundManager().AddRound(rnd)

	info := mixmessages.RoundInfo{
		ID: uint64(roundID),
	}

	slots, err := makePrecompTestBatch(instance, rnd, grp.GetP().ByteLen())
	if err != nil {
		t.Fatalf("Could not make mock slots: %v", err)
	}

	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          fakeHost,
	}
	mockStreamServer := newMockPrecompTestBatch(batchSize, slots)

	err = ReceivePrecompTestBatch(instance, &mockStreamServer, &info, auth)
	if err != nil {
		t.Fatalf("ReceivePrecompTestBatch error: %v", err)
	}
}

// Send over slots not conforming to expected data size. This will trigger
// the error case where slots are not of expected size (ie the size they
// would be received over realtime).
func TestReceivePrecompTestBatch_BadDataLength(t *testing.T) {
	instance, topology, grp := createMockInstance(t, 1, current.PRECOMPUTING)

	// Build a host around the last node
	lastNodeIndex := topology.Len() - 1
	lastNodeId := topology.GetNodeAtIndex(lastNodeIndex)
	params := connect.GetDefaultHostParams()
	params.MaxRetries = 0
	fakeHost, err := connect.NewHost(lastNodeId, "", nil, params)
	if err != nil {
		t.Errorf("Failed to create fakeHost, %s", err)
	}

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

	batchSize := 3
	rnd, err := round.New(grp, nil, roundID, []phase.Phase{p}, responseMap, topology, topology.GetNodeAtIndex(0), uint32(batchSize), instance.GetRngStreamGen(), nil, "0.0.0.0", nil, nil, nil, nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}

	instance.GetRoundManager().AddRound(rnd)

	info := mixmessages.RoundInfo{
		ID: uint64(roundID),
	}

	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          fakeHost,
	}

	emptySlots := make([]*mixmessages.Slot, 0)
	for i := 0; i < batchSize; i++ {
		emptySlots = append(emptySlots, &mixmessages.Slot{
			PayloadA: []byte("testBadSlot"),
		})
	}
	mockStreamServer := newMockPrecompTestBatch(batchSize, emptySlots)

	err = ReceivePrecompTestBatch(instance, &mockStreamServer, &info, auth)
	if err != nil && !strings.Contains(err.Error(), "incorrect data size received") {
		t.Fatalf("Expected error case, should not have received correct data size")
	}
}

// Send over slots not conforming to batch size.
func TestReceivePrecompTestBatch_BadSlotsLength(t *testing.T) {
	instance, topology, grp := createMockInstance(t, 1, current.PRECOMPUTING)

	// Build a host around the last node
	lastNodeIndex := topology.Len() - 1
	lastNodeId := topology.GetNodeAtIndex(lastNodeIndex)
	params := connect.GetDefaultHostParams()
	params.MaxRetries = 0
	fakeHost, err := connect.NewHost(lastNodeId, "", nil, params)
	if err != nil {
		t.Errorf("Failed to create fakeHost, %s", err)
	}

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

	batchSize := 3
	rnd, err := round.New(grp, nil, roundID, []phase.Phase{p}, responseMap, topology, topology.GetNodeAtIndex(0), uint32(batchSize), instance.GetRngStreamGen(), nil, "0.0.0.0", nil, nil, nil, nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}

	instance.GetRoundManager().AddRound(rnd)

	info := mixmessages.RoundInfo{
		ID: uint64(roundID),
	}

	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          fakeHost,
	}

	emptySlots := make([]*mixmessages.Slot, 0)
	for i := 0; i < batchSize; i++ {
		emptySlots = append(emptySlots, &mixmessages.Slot{
			PayloadA: []byte("testBadSlot"),
		})
	}
	// Send over the incorrect number of slots
	mockStreamServer := newMockPrecompTestBatch(batchSize/batchSize, emptySlots)

	err = ReceivePrecompTestBatch(instance, &mockStreamServer, &info, auth)
	if err != nil && !strings.Contains(err.Error(), "incorrect number of slots received") {
		t.Fatalf("Expected error case, should have received incorrect number of slots")
	}
}

// Send over bad auth.
func TestReceivePrecompTestBatch_BadAuth(t *testing.T) {
	instance, topology, grp := createMockInstance(t, 1, current.PRECOMPUTING)

	// Build a host around the first node
	firstNodeID := topology.GetNodeAtIndex(0)
	params := connect.GetDefaultHostParams()
	params.MaxRetries = 0
	firstNode, err := connect.NewHost(firstNodeID, "", nil, params)
	if err != nil {
		t.Errorf("Failed to create fakeHost, %s", err)
	}

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

	batchSize := 3
	rnd, err := round.New(grp, nil, roundID, []phase.Phase{p}, responseMap, topology, topology.GetNodeAtIndex(0), uint32(batchSize), instance.GetRngStreamGen(), nil, "0.0.0.0", nil, nil, nil, nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}

	instance.GetRoundManager().AddRound(rnd)

	info := mixmessages.RoundInfo{
		ID: uint64(roundID),
	}

	// Build incorrect auth object
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          firstNode,
	}

	emptySlots := make([]*mixmessages.Slot, 0)
	for i := 0; i < batchSize; i++ {
		emptySlots = append(emptySlots, &mixmessages.Slot{
			PayloadA: []byte("testBadSlot"),
		})
	}
	mockStreamServer := newMockPrecompTestBatch(batchSize, emptySlots)

	err = ReceivePrecompTestBatch(instance, &mockStreamServer, &info, auth)
	if err != nil && !strings.Contains(err.Error(), connect.AuthError(auth.Sender.GetId()).Error()) {
		t.Fatalf("Expected error case, should not have received correct data size")
	}
}

type testStreamPrecompTestBatch struct {
	batchSize int
	sent      int
	slots     []*mixmessages.Slot
}

func newMockPrecompTestBatch(batchSize int, slots []*mixmessages.Slot) testStreamPrecompTestBatch {
	return testStreamPrecompTestBatch{
		batchSize: batchSize,
		slots:     slots,
	}
}

func (t *testStreamPrecompTestBatch) SendAndClose(ack *messages.Ack) error {
	return nil
}

func (t *testStreamPrecompTestBatch) Recv() (*mixmessages.Slot, error) {
	t.sent++
	if t.sent == t.batchSize+1 {
		return nil, io.EOF
	}
	return t.slots[t.sent-1], nil
}

func (t testStreamPrecompTestBatch) SetHeader(md metadata.MD) error {
	return nil
}

func (t testStreamPrecompTestBatch) SendHeader(md metadata.MD) error {
	return nil
}

func (t testStreamPrecompTestBatch) SetTrailer(md metadata.MD) {
	return
}

func (t testStreamPrecompTestBatch) Context() context.Context {
	panic("implement me")
}

func (t testStreamPrecompTestBatch) SendMsg(m interface{}) error {
	return nil
}

func (t testStreamPrecompTestBatch) RecvMsg(m interface{}) error {
	return nil
}
