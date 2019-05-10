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
	"testing"
)

var receivedBatch *mixmessages.Batch

func TestPostRoundPublicKeyFunc(t *testing.T) {

	grp := initImplGroup()
	instance := server.CreateServerInstance(grp, &globals.UserMap{})

	batchSize := uint32(11)
	roundID := id.Round(0)

	mockPhase := initMockPhase()
	mockPhase.Ptype = phase.PrecompShare

	tagKey := phase.Type(phase.PrecompShare).String() + "Verification"
	responseMap := make(phase.ResponseMap)
	responseMap[tagKey] =
		phase.NewResponse(mockPhase.GetType(), mockPhase.GetType(),
			phase.Available)

	//responseMap[mockPhase.GetType().String()] =
	//	phase.NewResponse(mockPhase.GetType(), mockPhase.GetType(),
	//		phase.Available)

	topology := buildMockTopology(2)

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
	impl.Functions.PostRoundPublicKey(mockPk)

	if r.GetBuffer().CypherPublicKey.Cmp(grp.NewInt(42)) != 0 {
		///error here
	}

	// check receivedBatch

	//
	//for i := uint32(0); i < batchSize; i++ {
	//	mockBatch.Slots = append(mockBatch.Slots,
	//		&mixmessages.Slot{
	//			MessagePayload: []byte{byte(i)},
	//		})
	//}
	//
	//mockBatch.ForPhase = int32(mockPhase.GetType())
	//mockBatch.Round =

	//check the mock phase to see if the correct result has been stored
	//for index := range mockBatch.Slots {
	//	if mockPhase.chunks[index].Begin() != uint32(index) {
	//		t.Errorf("PostPhase: output chunk not equal to passed;"+
	//			"Expected: %v, Recieved: %v", index, mockPhase.chunks[index].Begin())
	//	}
	//
	//	if mockPhase.indices[index] != uint32(index) {
	//		t.Errorf("PostPhase: output index  not equal to passed;"+
	//			"Expected: %v, Recieved: %v", index, mockPhase.indices[index])
	//	}
	//}

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
	instance := server.CreateServerInstance(grp, &globals.UserMap{})

	batchSize := uint32(11)
	roundID := id.Round(0)

	mockPhase := initMockPhase()

	responseMap := make(phase.ResponseMap)
	responseMap[mockPhase.GetType().String()] =
		phase.NewResponse(mockPhase.GetType(), mockPhase.GetType(),
			phase.Available)

	topology := buildMockTopology(2)

	r := round.New(grp, roundID, []phase.Phase{mockPhase}, responseMap,
		topology, topology.GetNodeAtIndex(0), batchSize)

	instance.GetRoundManager().AddRound(r)

	// Build a mock public key
	mockRoundInfo := &mixmessages.RoundInfo{ID: uint64(roundID)}
	mockPk := &mixmessages.RoundPublicKey{
		Round: mockRoundInfo,
		Key:   []byte{1},
	}

	print(mockPk)

	// PostRoundPublicKeyFunc(instance, mockPostPhaseFunc)(mockPk)

}

func mockPostPhaseFunc(instance *server.Instance) func(message *mixmessages.Batch) {
	receivedBatch = &mixmessages.Batch{}
	return func(batch *mixmessages.Batch) {
		receivedBatch = batch
	}
}
