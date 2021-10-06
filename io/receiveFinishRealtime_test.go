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
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/testUtil"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/primitives/id"
	"google.golang.org/grpc/metadata"
	"io"
	"testing"
)

func TestReceiveFinishRealtime(t *testing.T) {
	instance, topology, grp := createMockInstance(t, 0, current.REALTIME)

	// Add nodes as hosts to topology
	for _, nid := range BuildMockNodeIDs(5, t) {
		h, _ := connect.NewHost(nid, "", nil, connect.GetDefaultHostParams())
		topology.AddHost(h)
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
	rnd, err := round.New(grp, roundID, []phase.Phase{p}, responseMap, topology, topology.GetNodeAtIndex(0), uint32(batchSize), instance.GetRngStreamGen(), nil, "0.0.0.0", nil, nil, nil, nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}

	instance.GetRoundManager().AddRound(rnd)

	info := mixmessages.RoundInfo{
		ID: uint64(roundID),
	}

	// Create a fake host and auth object to pass into function that needs it
	params := connect.GetDefaultHostParams()
	params.MaxRetries = 0
	fakeHost, err := connect.NewHost(topology.GetLastNode(), "", nil, params)
	if err != nil {
		t.Errorf("Failed to create fakeHost, %s", err)
	}
	auth := connect.Auth{
		IsAuthenticated: true,
		Sender:          fakeHost,
	}

	mockStreamServer := newMockFinishRealtimeStream(batchSize)

	err = ReceiveFinishRealtime(instance, &info, &mockStreamServer, &auth)

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

	batchSize := 3
	rnd, err := round.New(grp, roundID, []phase.Phase{p}, responseMap, topology, topology.GetNodeAtIndex(0), uint32(batchSize), instance.GetRngStreamGen(), nil, "0.0.0.0", nil, nil, nil, nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}

	instance.GetRoundManager().AddRound(rnd)

	info := mixmessages.RoundInfo{
		ID: uint64(roundID),
	}

	// Create a fake host and auth object to pass into function that needs it
	params := connect.GetDefaultHostParams()
	params.MaxRetries = 0
	fakeHost, err := connect.NewHost(topology.GetNodeAtIndex(0), "", nil, params)
	if err != nil {
		t.Errorf("Failed to create fakeHost, %s", err)
	}
	auth := connect.Auth{
		IsAuthenticated: false,
		Sender:          fakeHost,
	}

	mockStreamServer := newMockFinishRealtimeStream(batchSize)

	err = ReceiveFinishRealtime(instance, &info, &mockStreamServer, &auth)
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

	batchSize := 3
	rnd, err := round.New(grp, roundID, []phase.Phase{p}, responseMap, topology, topology.GetNodeAtIndex(0), uint32(batchSize), instance.GetRngStreamGen(), nil, "0.0.0.0", nil, nil, nil, nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}

	instance.GetRoundManager().AddRound(rnd)

	info := mixmessages.RoundInfo{
		ID: uint64(roundID),
	}

	// Create a fake host and auth object to pass into function that needs it
	newID := id.NewIdFromString("bad", id.Node, t)
	params := connect.GetDefaultHostParams()
	params.MaxRetries = 0
	fakeHost, err := connect.NewHost(newID, "", nil, params)
	if err != nil {
		t.Errorf("Failed to create fakeHost, %s", err)
	}
	auth := connect.Auth{
		IsAuthenticated: true,
		Sender:          fakeHost,
	}

	mockStreamServer := newMockFinishRealtimeStream(batchSize)

	err = ReceiveFinishRealtime(instance, &info, &mockStreamServer, &auth)
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

	// Add nodes as hosts to topology
	for _, nid := range BuildMockNodeIDs(5, t) {
		h, _ := connect.NewHost(nid, "", nil, connect.GetDefaultHostParams())
		topology.AddHost(h)
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
	rnd, err := round.New(grp, roundID, []phase.Phase{p}, responseMap, topology, topology.GetNodeAtIndex(0), uint32(batchSize), instance.GetRngStreamGen(), nil, "0.0.0.0", nil, nil, nil, nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}

	instance.GetRoundManager().AddRound(rnd)

	info := mixmessages.RoundInfo{
		ID: uint64(roundID),
	}

	// Create a fake host and auth object to pass into function that needs it
	params := connect.GetDefaultHostParams()
	params.MaxRetries = 0
	fakeHost, err := connect.NewHost(topology.GetNodeAtIndex(topology.Len()-1),
		"", nil, params)
	if err != nil {
		t.Errorf("Failed to create fakeHost, %s", err)
	}
	auth := connect.Auth{
		IsAuthenticated: true,
		Sender:          fakeHost,
	}

	mockStreamServer := newMockFinishRealtimeStream(batchSize)

	err = ReceiveFinishRealtime(instance, &info, &mockStreamServer, &auth)

	if err != nil {
		t.Errorf("ReceiveFinishRealtime: errored: %+v", err)
	}

}

type testStreamFinishRealtime struct {
	batchSize int
	sent      int
}

func newMockFinishRealtimeStream(batchSize int) testStreamFinishRealtime {
	return testStreamFinishRealtime{
		batchSize: batchSize,
	}
}

func (t testStreamFinishRealtime) SendAndClose(ack *messages.Ack) error {
	return nil
}

func (t *testStreamFinishRealtime) Recv() (*mixmessages.Slot, error) {
	t.sent++
	if t.sent == t.batchSize+1 {
		return nil, io.EOF
	}
	return &mixmessages.Slot{}, nil
}

func (t testStreamFinishRealtime) SetHeader(md metadata.MD) error {
	return nil
}

func (t testStreamFinishRealtime) SendHeader(md metadata.MD) error {
	return nil
}

func (t testStreamFinishRealtime) SetTrailer(md metadata.MD) {
	return
}

func (t testStreamFinishRealtime) Context() context.Context {
	return nil
}

func (t testStreamFinishRealtime) SendMsg(m interface{}) error {
	return nil
}

func (t testStreamFinishRealtime) RecvMsg(m interface{}) error {
	return nil
}
