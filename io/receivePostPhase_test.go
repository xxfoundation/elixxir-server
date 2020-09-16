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
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/internal/state"
	"gitlab.com/elixxir/server/testUtil"
	"gitlab.com/xx_network/comms/connect"
	"testing"
	"time"
)

func TestNewImplementation_PostPhase(t *testing.T) {
	batchSize := uint32(11)
	roundID := id.Round(0)
	grp := initImplGroup()

	topology := connect.NewCircuit(BuildMockNodeIDs(2, t))

	def := internal.Definition{
		UserRegistry:    &globals.UserMap{},
		ResourceMonitor: &measure.ResourceMonitor{},
		FullNDF:         testUtil.NDF,
		PartialNDF:      testUtil.NDF,
		Flags:           internal.Flags{DisableIpOverride: true},
	}

	def.ID = topology.GetNodeAtIndex(0)
	def.Gateway.ID = def.ID.DeepCopy()
	def.Gateway.ID.SetType(id.Gateway)

	m := state.NewTestMachine(dummyStates, current.PRECOMPUTING, t)
	instance, _ := internal.CreateServerInstance(&def, NewImplementation, m,
		"1.1.0")

	mockPhase := testUtil.InitMockPhase(t)

	responseMap := make(phase.ResponseMap)
	responseMap[mockPhase.GetType().String()] =
		phase.NewResponse(phase.ResponseDefinition{mockPhase.GetType(),
			[]phase.State{phase.Active}, mockPhase.GetType()})

	r, err := round.New(grp, &globals.UserMap{}, roundID, []phase.Phase{mockPhase},
		responseMap, topology, topology.GetNodeAtIndex(0), batchSize,
		instance.GetRngStreamGen(), nil, "0.0.0.0", nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}

	instance.GetRoundManager().AddRound(r)
	err = instance.GetStateMachine().Start()
	if err != nil {
		t.Errorf("Failed to run instance: %+v", err)
		return
	}
	// get the impl
	impl := NewImplementation(instance)

	// Build a mock mockBatch to receive
	mockBatch := &mixmessages.Batch{}

	for i := uint32(0); i < batchSize; i++ {
		mockBatch.Slots = append(mockBatch.Slots,
			&mixmessages.Slot{
				PayloadA: []byte{byte(i)},
			})
	}

	mockBatch.FromPhase = int32(mockPhase.GetType())
	mockBatch.Round = &mixmessages.RoundInfo{ID: uint64(roundID)}

	// Build a host around the last node
	lastNodeIndex := topology.Len() - 1
	lastNodeId := topology.GetNodeAtIndex(lastNodeIndex)
	params := connect.GetDefaultHostParams()
	params.MaxRetries = 0
	fakeHost, err := connect.NewHost(lastNodeId, "", nil, params)
	if err != nil {
		t.Errorf("Failed to create fakeHost, %s", err)
	}

	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          fakeHost,
	}
	// send the mockBatch to the impl
	err = impl.PostPhase(mockBatch, auth)
	if err != nil {
		t.Errorf("Failed to PostPhase: %+v", err)
	}

	//check the mock phase to see if the correct result has been stored
	for index := range mockBatch.Slots {
		if mockPhase.GetChunks()[index].Begin() != uint32(index) {
			t.Errorf("PostPhase: output chunk not equal to passed;"+
				"Expected: %v, Received: %v", index, mockPhase.GetChunks()[index].Begin())
		}

		if mockPhase.GetIndices()[index] != uint32(index) {
			t.Errorf("PostPhase: output index  not equal to passed;"+
				"Expected: %v, Received: %v", index, mockPhase.GetIndices()[index])
		}
	}

	var queued bool
	timer := time.NewTimer(time.Second)
	select {
	case <-instance.GetResourceQueue().GetQueue(t):
		queued = true
	case <-timer.C:
		queued = false
	}

	if !queued {
		t.Errorf("PostPhase: The phase was not queued properly")
	}
}

// Happy path
func TestPostPhase_NoAuth(t *testing.T) {
	// Defer to a success when PostPhase call panics
	defer func() {
		if r := recover(); r != nil {
			return
		}
	}()

	batchSize := uint32(11)
	roundID := id.Round(0)

	grp := initImplGroup()
	instance, topology := mockServerInstance(t, current.PRECOMPUTING)
	rnd, err := round.New(grp, nil, id.Round(0), make([]phase.Phase, 0),
		make(phase.ResponseMap), topology, topology.GetNodeAtIndex(0),
		3, instance.GetRngStreamGen(), nil, "0.0.0.0", nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}
	instance.GetRoundManager().AddRound(rnd)

	mockPhase := testUtil.InitMockPhase(t)

	// Build a mock mockBatch to receive
	mockBatch := &mixmessages.Batch{}

	for i := uint32(0); i < batchSize; i++ {
		mockBatch.Slots = append(mockBatch.Slots,
			&mixmessages.Slot{
				Index:    i,
				PayloadA: []byte{byte(i)},
			})
	}

	mockBatch.FromPhase = int32(mockPhase.GetType())
	mockBatch.Round = &mixmessages.RoundInfo{ID: uint64(roundID)}

	// Make an auth object around the last node
	lastNodeIndex := topology.Len() - 1
	lastNodeId := topology.GetNodeAtIndex(lastNodeIndex)
	params := connect.GetDefaultHostParams()
	params.MaxRetries = 0
	fakeHost, err := connect.NewHost(lastNodeId, "", nil, params)
	if err != nil {
		t.Errorf("Failed to create fakeHost, %s", err)
	}

	auth := &connect.Auth{
		IsAuthenticated: false,
		Sender:          fakeHost,
	}

	err = ReceivePostPhase(mockBatch, instance, auth)
	if err == nil {
		t.Errorf("Expected error case, should not be able to ReceivePostPhase when not authenticated")
	}
}

//Error path
func TestPostPhase_WrongSender(t *testing.T) { // Defer to a success when PostPhase call panics
	defer func() {
		if r := recover(); r != nil {
			return
		}
	}()

	batchSize := uint32(11)
	roundID := id.Round(0)

	instance, topology := mockServerInstance(t, current.PRECOMPUTING)
	mockPhase := testUtil.InitMockPhase(t)

	// Build a mock mockBatch to receive
	mockBatch := &mixmessages.Batch{}

	for i := uint32(0); i < batchSize; i++ {
		mockBatch.Slots = append(mockBatch.Slots,
			&mixmessages.Slot{
				Index:    i,
				PayloadA: []byte{byte(i)},
			})
	}

	mockBatch.FromPhase = int32(mockPhase.GetType())
	mockBatch.Round = &mixmessages.RoundInfo{ID: uint64(roundID)}

	// Make an auth object around a node that is not the previous node
	lastNodeIndex := topology.Len() - 2
	lastNodeId := topology.GetNodeAtIndex(lastNodeIndex)
	params := connect.GetDefaultHostParams()
	params.MaxRetries = 0
	fakeHost, err := connect.NewHost(lastNodeId, "", nil, params)
	if err != nil {
		t.Errorf("Failed to create fakeHost, %s", err)
	}

	auth := &connect.Auth{
		IsAuthenticated: false,
		Sender:          fakeHost,
	}

	err = ReceivePostPhase(mockBatch, instance, auth)
	if err == nil {
		t.Errorf("Expected error case, should not be able to ReceivePostPhase when not authenticated")
	}
}

// Error Path
func TestStreamPostPhase_NoAuth(t *testing.T) {
	batchSize := uint32(11)
	roundID := id.Round(0)

	instance, topology := mockServerInstance(t, current.PRECOMPUTING)
	mockPhase := testUtil.InitMockPhase(t)

	// Build a mock mockBatch to receive
	mockBatch := &mixmessages.Batch{}

	for i := uint32(0); i < batchSize; i++ {
		mockBatch.Slots = append(mockBatch.Slots,
			&mixmessages.Slot{
				Index:    i,
				PayloadA: []byte{byte(i)},
			})
	}

	mockBatch.FromPhase = int32(mockPhase.GetType())
	mockBatch.Round = &mixmessages.RoundInfo{ID: uint64(roundID)}

	mockStreamServer := MockStreamPostPhaseServer{
		batch: mockBatch,
	}

	// Make an auth object around the last node
	lastNodeIndex := topology.Len() - 1
	lastNodeId := topology.GetNodeAtIndex(lastNodeIndex)
	params := connect.GetDefaultHostParams()
	params.MaxRetries = 0
	fakeHost, err := connect.NewHost(lastNodeId, "", nil, params)
	if err != nil {
		t.Errorf("Failed to create fakeHost, %s", err)
	}

	auth := &connect.Auth{
		IsAuthenticated: false,
		Sender:          fakeHost,
	}

	err = ReceiveStreamPostPhase(mockStreamServer, instance, auth)

	if err != nil {
		return
	}

	t.Errorf("Expected error case, should not be able to ReceiveStreamPostPhase when not authenticated")
}

// Error path
func TestStreamPostPhase_WrongSender(t *testing.T) {
	batchSize := uint32(11)
	roundID := id.Round(0)

	instance, topology := mockServerInstance(t, current.PRECOMPUTING)
	mockPhase := testUtil.InitMockPhase(t)

	// Build a mock mockBatch to receive
	mockBatch := &mixmessages.Batch{}

	for i := uint32(0); i < batchSize; i++ {
		mockBatch.Slots = append(mockBatch.Slots,
			&mixmessages.Slot{
				Index:    i,
				PayloadA: []byte{byte(i)},
			})
	}

	mockBatch.FromPhase = int32(mockPhase.GetType())
	mockBatch.Round = &mixmessages.RoundInfo{ID: uint64(roundID)}

	mockStreamServer := MockStreamPostPhaseServer{
		batch: mockBatch,
	}

	// Make an auth object around a non previous node
	lastNodeIndex := topology.Len() - 2
	lastNodeId := topology.GetNodeAtIndex(lastNodeIndex)
	params := connect.GetDefaultHostParams()
	params.MaxRetries = 0
	fakeHost, err := connect.NewHost(lastNodeId, "", nil, params)
	if err != nil {
		t.Errorf("Failed to create fakeHost, %s", err)
	}

	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          fakeHost,
	}

	err = ReceiveStreamPostPhase(mockStreamServer, instance, auth)

	if err != nil {
		return
	}

	t.Errorf("Expected error case, should not be able to ReceiveStreamPostPhase when not authenticated")

}

func TestNewImplementation_StreamPostPhase(t *testing.T) {
	batchSize := uint32(11)
	roundID := id.Round(0)

	instance, topology, grp := createMockInstance(t, 1, current.PRECOMPUTING)

	mockPhase := testUtil.InitMockPhase(t)

	responseMap := make(phase.ResponseMap)
	responseMap[mockPhase.GetType().String()] =
		phase.NewResponse(phase.ResponseDefinition{mockPhase.GetType(),
			[]phase.State{phase.Active}, mockPhase.GetType()})

	r, err := round.New(grp, &globals.UserMap{}, roundID, []phase.Phase{mockPhase},
		responseMap, topology, topology.GetNodeAtIndex(0), batchSize,
		instance.GetRngStreamGen(), nil, "0.0.0.0", nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}

	instance.GetRoundManager().AddRound(r)

	// get the impl
	impl := NewImplementation(instance)

	// Build a mock mockBatch to receive
	mockBatch := &mixmessages.Batch{}

	for i := uint32(0); i < batchSize; i++ {
		mockBatch.Slots = append(mockBatch.Slots,
			&mixmessages.Slot{
				Index:    i,
				PayloadA: []byte{byte(i)},
			})
	}

	mockBatch.FromPhase = int32(mockPhase.GetType())
	mockBatch.Round = &mixmessages.RoundInfo{ID: uint64(roundID)}

	mockStreamServer := MockStreamPostPhaseServer{
		batch: mockBatch,
	}

	// Make an auth object around the last node
	lastNodeId := topology.GetPrevNode(instance.GetID())
	params := connect.GetDefaultHostParams()
	params.MaxRetries = 0
	fakeHost, err := connect.NewHost(lastNodeId, "", nil, params)
	if err != nil {
		t.Errorf("Failed to create fakeHost, %s", err)
	}

	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          fakeHost,
	}

	//send the mockBatch to the impl
	err = impl.StreamPostPhase(mockStreamServer, auth)

	if err != nil {
		t.Errorf("StreamPostPhase: error on call: %+v",
			err)
	}

	//check the mock phase to see if the correct result has been stored
	for index := range mockBatch.Slots {
		if mockPhase.GetChunks()[index].Begin() != uint32(index) {
			t.Errorf("StreamPostPhase: output chunk not equal to passed;"+
				"Expected: %v, Received: %v", index, mockPhase.GetChunks()[index].Begin())
		}

		if mockPhase.GetIndices()[index] != uint32(index) {
			t.Errorf("StreamPostPhase: output index  not equal to passed;"+
				"Expected: %v, Received: %v", index, mockPhase.GetIndices()[index])
		}
	}

	var queued bool

	select {
	case <-instance.GetResourceQueue().GetQueue(t):
		queued = true
	default:
		queued = false
	}

	if !queued {
		t.Errorf("StreamPostPhase: The phase was not queued properly")
	}
}
