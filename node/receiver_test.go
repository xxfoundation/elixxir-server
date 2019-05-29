////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package node

import (
	"fmt"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/graphs/realtime"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/services"
	"reflect"
	"runtime"
	"testing"
	"time"
)

func TestReceivePostNewBatch_Errors(t *testing.T) {
	// This round should be at a state where its precomp is complete.
	// So, we might want more than one phase,
	// since it's at a boundary between phases.
	grp := initImplGroup()
	topology := buildMockTopology(1)
	instance := server.CreateServerInstance(grp, topology.GetNodeAtIndex(0),
		&globals.UserMap{})
	instance.InitFirstNode()

	const batchSize = 1
	const roundID = 2

	// Does the mockPhase move through states?
	precompReveal := initMockPhase()
	precompReveal.Ptype = phase.PrecompReveal
	realDecrypt := initMockPhase()
	realDecrypt.Ptype = phase.RealDecrypt

	tagKey := realDecrypt.Ptype.String()
	responseMap := make(phase.ResponseMap)
	responseMap[tagKey] =
		phase.NewResponse(realDecrypt.GetType(), realDecrypt.GetType(),
			phase.Available)

	// Well, this round needs to at least be on the precomp queue?
	// If it's not on the precomp queue,
	// that would let us test the error being returned.
	r := round.New(grp, instance.GetUserRegistry(), roundID,
		[]phase.Phase{precompReveal, realDecrypt},
		responseMap, topology, topology.GetNodeAtIndex(0), batchSize)
	instance.GetRoundManager().AddRound(r)

	// Build a fake batch for the reception handler
	// This emulates what the gateway would send to the comm
	batch := &mixmessages.Batch{
		Round: &mixmessages.RoundInfo{
			ID: roundID,
		},
		ForPhase: int32(phase.RealDecrypt),
		Slots: []*mixmessages.Slot{
			{
				// Do the fields need to be populated?
				SenderID:       nil,
				MessagePayload: nil,
				AssociatedData: nil,
				Salt:           nil,
				KMACs:          nil,
			},
		},
	}

	err := ReceivePostNewBatch(instance, batch)
	if err == nil {
		t.Error("ReceivePostNewBatch should have errored out if there were no" +
			" precomputations available")
	}

	// OK, let's put that round on the queue of completed precomps now,
	// which should cause the reception handler to function normally.
	// This should panic because the expected states aren't populated correctly,
	// so the realtime can't continue to be processed.
	defer func() {
		panicResult := recover()
		panicString := panicResult.(string)
		if panicString == "" {
			t.Error("There was no panicked error from the HandleIncomingComm" +
				" call")
		}
	}()
	instance.GetCompletedPrecomps().Push(r)
	err = ReceivePostNewBatch(instance, batch)
}

// Tests the happy path of ReceivePostNewBatch, demonstrating that it can start
// realtime processing with a new batch from the gateway.
func TestReceivePostNewBatch(t *testing.T) {
	// This round should be at a state where its precomp is complete.
	// So, we might want more than one phase,
	// since it's at a boundary between phases.
	grp := initImplGroup()
	topology := buildMockTopology(1)
	instance := server.CreateServerInstance(grp, topology.GetNodeAtIndex(0),
		&globals.UserMap{})
	instance.InitFirstNode()

	const batchSize = 1
	const roundID = 2

	PanicHandler := func(g, m string, err error) {
		panic(fmt.Sprintf("Error in module %s of graph %s: %s", g, m, err.Error()))
	}
	gg := services.NewGraphGenerator(4, PanicHandler, uint8(runtime.NumCPU()),
		1, 1.0)

	// Need to override realDecrypt's graph to make sure the correct data gets
	// passed to the phase
	// Can I make a real, actual phase here?
	// That's the next least effort thing to get the test working...
	//realDecrypt := initMockPhase()
	//realDecrypt.Ptype = phase.RealDecrypt
	realDecrypt := phase.New(realtime.InitDecryptGraph(gg), phase.RealDecrypt,
		func(network *node.NodeComms, batchSize uint32, roundID id.Round, phaseTy phase.Type, getChunk phase.GetChunk, getMessage phase.GetMessage, topology *circuit.Circuit, nodeId *id.Node) error {
			return nil
		}, 5*time.Second)

	tagKey := realDecrypt.GetType().String()
	responseMap := make(phase.ResponseMap)
	// So, we can just make the responseMap accept all possible phase states,
	// right? This shouldn't be use as a guideline for real usage, but should
	// at least allow the test to pass.
	// TODO Should move this back to the one state that it should be in for
	//  the response on this node. Don't remember what that is.
	responseMap[tagKey] =
		phase.NewResponse(realDecrypt.GetType(), realDecrypt.GetType(),
			phase.Available, phase.Verified, phase.Queued, phase.Computed,
			phase.Running, phase.Initialized)

	// Well, this round needs to at least be on the precomp queue?
	// If it's not on the precomp queue,
	// that would let us test the error being returned.
	r := round.New(grp, instance.GetUserRegistry(), roundID,
		[]phase.Phase{realDecrypt}, responseMap, topology,
		topology.GetNodeAtIndex(0), batchSize)
	instance.GetRoundManager().AddRound(r)
	instance.GetCompletedPrecomps().Push(r)

	// Build a fake batch for the reception handler
	// This emulates what the gateway would send to the comm
	batch := &mixmessages.Batch{
		Round: &mixmessages.RoundInfo{
			ID: roundID,
		},
		ForPhase: int32(phase.RealDecrypt),
		Slots: []*mixmessages.Slot{
			{
				// Do the fields need to be populated?
				// Yes, but only to check if the batch made it to the phase
				SenderID:       []byte{1},
				MessagePayload: []byte{2},
				AssociatedData: []byte{3},
				Salt:           []byte{4},
				KMACs:          [][]byte{{5}},
			},
		},
	}
	err := ReceivePostNewBatch(instance, batch)
	if err != nil {
		t.Error(err)
	}
	// This should automatically cause the test to succeed if SetState has
	// been implemented correctly in the mock phase
	//realDecrypt.AttemptTransitionToQueued()
	// It did not work. This means we either need to upgrade the mock phase
	// with more functionality, or use a real phase.

	// We verify that the Realtime Decrypt phase has been enqueued, and that the
	// batch running in the phase has the correct data
	if realDecrypt.GetState() != phase.Queued {
		t.Errorf("Realtime decrypt states was %v, not %v",
			realDecrypt.GetState(), phase.Queued)
	}
}

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
	r := round.New(grp, instance.GetUserRegistry(), roundID,
		[]phase.Phase{mockPhase}, responseMap,
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
	r := round.New(grp, instance.GetUserRegistry(), roundID,
		[]phase.Phase{mockPhase}, responseMap,
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
	instance.GetRoundManager().AddRound(round.New(grp,
		instance.GetUserRegistry(), roundID,
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
		instances[i].GetRoundManager().AddRound(round.New(grp,
			instances[i].GetUserRegistry(), roundID,
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
