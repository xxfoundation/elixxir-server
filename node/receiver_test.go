////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package node

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/server/round"
	"reflect"
	"testing"
)

var receivedBatch *mixmessages.Batch

func TestPostRoundPublicKeyFunc(t *testing.T) {

	grp := initImplGroup()
	topology := buildMockTopology(5)

	instance := server.CreateServerInstance(grp, topology.GetNodeAtIndex(1), &globals.UserMap{})

	batchSize := uint32(11)
	roundID := id.Round(0)

	mockPhase := initMockPhase()
	mockPhase.Ptype = phase.PrecompShare

	tagKey := mockPhase.GetType().String() + "Verification"
	responseMap := make(phase.ResponseMap)
	responseMap[tagKey] =
		phase.NewResponse(mockPhase.GetType(), mockPhase.GetType(),
			phase.Available)

	// Skip first node
	r := round.New(grp, roundID, []phase.Phase{mockPhase}, responseMap,
		topology, topology.GetNodeAtIndex(1), batchSize)

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
	impl.Functions.PostPhase = func(message *mixmessages.Batch) {
		actualBatch = message
	}

	impl.Functions.PostRoundPublicKey(mockPk)

	// Verify that a PostPhase isn't called by ensuring callback
	// doesn't set the actual by comparing it to the empty batch
	if !batchEq(actualBatch, emptyBatch) {
		t.Errorf("Actual batch was not equal to empty batch in mock postphase")
	}

	if r.GetBuffer().CypherPublicKey.Cmp(grp.NewInt(42)) != 0 {
		// Error here
		t.Errorf("CypherPublicKey doesn't match expected value of the public key")
	}

	var queued bool

	select {
	case <-instance.GetResourceQueue().GetQueue():
		queued = true
	default:
		queued = false
	}

	if !queued {
		t.Errorf("PostPhase: The phase was not queued properly")
	}
}

func TestPostRoundPublicKeyFunc_FirstNodeSendsBatch(t *testing.T) {

	grp := initImplGroup()
	topology := buildMockTopology(5)

	instance := server.CreateServerInstance(grp, topology.GetNodeAtIndex(0), &globals.UserMap{})

	batchSize := uint32(11)
	roundID := id.Round(0)

	mockPhase := initMockPhase()
	mockPhase.Ptype = phase.PrecompShare

	tagKey := mockPhase.GetType().String() + "Verification"
	responseMap := make(phase.ResponseMap)
	responseMap[tagKey] =
		phase.NewResponse(mockPhase.GetType(), mockPhase.GetType(),
			phase.Available)

	// Don't skip first node
	r := round.New(grp, roundID, []phase.Phase{mockPhase}, responseMap,
		topology, topology.GetNodeAtIndex(0), batchSize)

	instance.GetRoundManager().AddRound(r)

	// Build a mock public key
	mockRoundInfo := &mixmessages.RoundInfo{ID: uint64(roundID)}
	mockPk := &mixmessages.RoundPublicKey{
		Round: mockRoundInfo,
		Key:   []byte{42},
	}

	impl := NewImplementation(instance)

	actualBatch := &mixmessages.Batch{}
	expectedBatch := &mixmessages.Batch{}

	// Create expected batch
	expectedBatch.Round = mockPk.Round
	expectedBatch.ForPhase = int32(phase.PrecompDecrypt)
	expectedBatch.Slots = make([]*mixmessages.Slot, batchSize)

	for i := uint32(0); i < batchSize; i++ {
		expectedBatch.Slots[i] = &mixmessages.Slot{
			EncryptedMessageKeys:            []byte{1},
			EncryptedAssociatedDataKeys:     []byte{1},
			PartialMessageCypherText:        []byte{1},
			PartialAssociatedDataCypherText: []byte{1},
		}
	}

	impl.Functions.PostPhase = func(message *mixmessages.Batch) {
		actualBatch = message
	}

	impl.Functions.PostRoundPublicKey(mockPk)

	// Verify that a PostPhase is called by ensuring callback
	// does set the actual by comparing it to the expected batch
	if !batchEq(actualBatch, expectedBatch) {
		t.Errorf("Expected batch was not equal to actual batch in mock postphase")
	}

	if r.GetBuffer().CypherPublicKey.Cmp(grp.NewInt(42)) != 0 {
		// Error here
		t.Errorf("CypherPublicKey doesn't match expected value of the public key")
	}

	var queued bool

	select {
	case <-instance.GetResourceQueue().GetQueue():
		queued = true
	default:
		queued = false
	}

	if !queued {
		t.Errorf("PostPhase: The phase was not queued properly")
	}

}

// batchEq compares two batches to see if they are equal
// Return true if they are equal and false otherwise
func batchEq(a *mixmessages.Batch, b *mixmessages.Batch) bool {
	if a.GetRound() != b.GetRound() {
		return false
	}

	if a.GetForPhase() != b.GetForPhase() {
		return false
	}

	if len(a.GetSlots()) != len(b.GetSlots()) {
		return false
	}

	bSlots := b.GetSlots()
	for index, slot := range a.GetSlots() {
		if !reflect.DeepEqual(slot, bSlots[index]) {
			return false
		}
	}

	return true
}

// Shows that ReceivePostPrecompResult panics when the round isn't in
// the round manager
func TestPostPrecompResultFunc_Error_NoRound(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("There was no panic when an invalid round was passed")
		}
	}()
	grp := initImplGroup()
	topology := buildMockTopology(5)

	instance := server.CreateServerInstance(grp, topology.GetNodeAtIndex(0), &globals.UserMap{})
	// We haven't set anything up,
	// so this should panic because the round can't be found
	err := ReceivePostPrecompResult(instance, 0, []*mixmessages.Slot{})

	if err == nil {
		t.Error("Didn't get an error from a nonexistent round")
	}
}

// Shows that ReceivePostPrecompResult returns an error when there are a wrong
// number of slots in the message
func TestPostPrecompResultFunc_Error_WrongNumSlots(t *testing.T) {
	// Smoke tests the management part of PostPrecompResult
	grp := initImplGroup()
	topology := buildMockTopology(5)

	instance := server.CreateServerInstance(grp, topology.GetNodeAtIndex(0), &globals.UserMap{})
	roundID := id.Round(45)
	// Is this the right setup for the response?
	response := phase.NewResponse(phase.PrecompReveal, phase.PrecompReveal,
		phase.Available)
	responseMap := make(phase.ResponseMap)
	responseMap[phase.PrecompReveal.String()+"Verification"] = response
	// This is quite a bit of setup...
	p := initMockPhase()
	p.Ptype = phase.PrecompReveal
	instance.GetRoundManager().AddRound(round.New(grp, roundID,
		[]phase.Phase{p}, responseMap,
		topology, topology.GetNodeAtIndex(0), 3))
	// This should give an error because we give it fewer slots than are in the
	// batch
	err := ReceivePostPrecompResult(instance, uint64(roundID), []*mixmessages.Slot{})

	if err == nil {
		t.Error("Didn't get an error from the wrong number of slots")
	}
}

// Shows that PostPrecompResult puts the completed precomputation on the
// channel on the first node when it has valid data
// Shows that PostPrecompResult doesn't result in errors on the other nodes
func TestPostPrecompResultFunc(t *testing.T) {
	// Smoke tests the management part of PostPrecompResult
	grp := initImplGroup()
	const numNodes = 5
	topology := buildMockTopology(numNodes)

	// Set up all the instances
	var instances []*server.Instance
	for i := 0; i < numNodes; i++ {
		instances = append(instances, server.CreateServerInstance(
			grp, topology.GetNodeAtIndex(i), &globals.UserMap{}))
	}
	instances[0].InitFirstNode()

	// Set up a round on all the instances
	roundID := id.Round(45)
	for i := 0; i < numNodes; i++ {
		response := phase.NewResponse(phase.PrecompReveal, phase.PrecompReveal,
			phase.Available)
		responseMap := make(phase.ResponseMap)
		responseMap[phase.PrecompReveal.String()+"Verification"] = response
		// This is quite a bit of setup...
		p := initMockPhase()
		p.Ptype = phase.PrecompReveal
		instances[i].GetRoundManager().AddRound(round.New(grp, roundID,
			[]phase.Phase{p}, responseMap, topology, topology.GetNodeAtIndex(i), 3))
	}

	// Initially, there should be zero rounds on the precomp queue
	if len(instances[0].GetCompletedPrecomps().CompletedPrecomputations) != 0 {
		t.Error("Expected completed precomps to be empty")
	}

	// Since we give this 3 slots with the correct fields populated,
	// it should work without errors on all nodes
	for i := 0; i < numNodes; i++ {
		err := ReceivePostPrecompResult(instances[i], uint64(roundID),
			[]*mixmessages.Slot{{
				PartialMessageCypherText:        grp.NewInt(3).Bytes(),
				PartialAssociatedDataCypherText: grp.NewInt(4).Bytes(),
			}, {
				PartialMessageCypherText:        grp.NewInt(3).Bytes(),
				PartialAssociatedDataCypherText: grp.NewInt(4).Bytes(),
			}, {
				PartialMessageCypherText:        grp.NewInt(3).Bytes(),
				PartialAssociatedDataCypherText: grp.NewInt(4).Bytes(),
			}})

		if err != nil {
			t.Errorf("Error posting precomp on node %v: %v", i, err)
		}
	}

	// Then, after the reception handler ran successfully,
	// there should be 1 precomputation in the buffer on the first node
	// The others don't have this variable initialized
	if len(instances[0].GetCompletedPrecomps().CompletedPrecomputations) != 1 {
		t.Error("Expected completed precomps to have the one precomp we posted")
	}
}
