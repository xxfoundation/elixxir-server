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

	tagKey := phase.Type(phase.PrecompShare).String() + "Verification"
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

	if !batchEq(actualBatch, emptyBatch) {
		t.Errorf("")
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

	instance := server.CreateServerInstance(grp, topology.GetNodeAtIndex(1), &globals.UserMap{})

	batchSize := uint32(11)
	roundID := id.Round(0)

	mockPhase := initMockPhase()
	mockPhase.Ptype = phase.PrecompShare

	tagKey := phase.Type(phase.PrecompShare).String() + "Verification"
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
	impl.Functions.PostRoundPublicKey(mockPk)

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

func mockPostPhaseFunc(instance *server.Instance) func(message *mixmessages.Batch) {
	receivedBatch = &mixmessages.Batch{}
	return func(batch *mixmessages.Batch) {
		receivedBatch = batch
	}
}

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