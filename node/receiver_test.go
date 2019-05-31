////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package node

import (
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/graphs/realtime"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/conf"
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
	instance := server.CreateServerInstance(conf.Params{
		Groups: conf.Groups{
			CMix: grp,
		},
		NodeID: topology.GetNodeAtIndex(0),
	}, &globals.UserMap{})
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
	responseMap[tagKey] = phase.NewResponse(phase.ResponseDefinition{
		PhaseAtSource:  realDecrypt.GetType(),
		PhaseToExecute: realDecrypt.GetType(),
		ExpectedStates: []phase.State{phase.Available},
	})

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
// Note: In this case, the happy path includes an error from one of the slots
// that has cryptographically incorrect data.
func TestReceivePostNewBatch(t *testing.T) {
	grp := initImplGroup()
	topology := buildMockTopology(1)
	registry := &globals.UserMap{}
	instance := server.CreateServerInstance(conf.Params{
		Groups: conf.Groups{
			CMix: grp,
		},
		NodeID: topology.GetNodeAtIndex(0),
	}, registry)
	instance.InitFirstNode()

	// Make and register a user
	sender := registry.NewUser(grp)
	registry.UpsertUser(sender)

	const batchSize = 1
	const roundID = 2

	PanicHandler := func(g, m string, err error) {
		panic(fmt.Sprintf("Error in module %s of graph %s: %s", g, m, err.Error()))
	}
	gg := services.NewGraphGenerator(4, PanicHandler, uint8(runtime.NumCPU()),
		1, 1.0)

	realDecrypt := phase.New(phase.Definition{
		Graph: realtime.InitDecryptGraph(gg),
		Type:  phase.RealDecrypt,
		TransmissionHandler: func(network *node.NodeComms, batchSize uint32, roundID id.Round, phaseTy phase.Type, getChunk phase.GetChunk, getMessage phase.GetMessage, topology *circuit.Circuit, nodeId *id.Node) error {
			return nil
		},
		Timeout:        5 * time.Second,
		DoVerification: false,
	})

	tagKey := realDecrypt.GetType().String()
	responseMap := make(phase.ResponseMap)
	responseMap[tagKey] = phase.NewResponse(phase.ResponseDefinition{
		PhaseAtSource:  realDecrypt.GetType(),
		ExpectedStates: []phase.State{phase.Available},
		PhaseToExecute: realDecrypt.GetType(),
	})

	// We need this round to be on the precomp queue
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
				SenderID:       sender.ID.Bytes(),
				MessagePayload: []byte{2},
				AssociatedData: []byte{3},
				// Because the salt is just one byte,
				// this should fail in the Realtime Decrypt graph.
				Salt:  []byte{4},
				KMACs: [][]byte{{5}},
			},
		},
	}
	// Actually, this should return an error because the batch has a malformed
	// slot in it, so once we implement per-slot errors we can test all the
	// realtime decrypt error cases from this reception handler if we want
	err := ReceivePostNewBatch(instance, batch)
	if err != nil {
		t.Error(err)
	}

	// We verify that the Realtime Decrypt phase has been enqueued
	if realDecrypt.GetState() != phase.Queued {
		t.Errorf("Realtime decrypt states was %v, not %v",
			realDecrypt.GetState(), phase.Queued)
	}
}

func TestNewImplementation_PostPhase(t *testing.T) {
	batchSize := uint32(11)
	roundID := id.Round(0)

	grp := initImplGroup()
	nid := server.GenerateId()
	params := conf.Params{
		Groups: conf.Groups{CMix: grp},
		NodeID: nid,
	}
	instance := server.CreateServerInstance(params, &globals.UserMap{})
	mockPhase := initMockPhase()

	responseMap := make(phase.ResponseMap)
	responseMap[mockPhase.GetType().String()] =
		phase.NewResponse(phase.ResponseDefinition{mockPhase.GetType(),
			[]phase.State{phase.Available}, mockPhase.GetType()})

	topology := buildMockTopology(2)

	r := round.New(grp, &globals.UserMap{}, roundID, []phase.Phase{mockPhase},
		responseMap, topology, topology.GetNodeAtIndex(0), batchSize)

	instance.GetRoundManager().AddRound(r)

	// get the impl
	impl := NewImplementation(instance)

	// Build a mock mockBatch to receive
	mockBatch := &mixmessages.Batch{}

	for i := uint32(0); i < batchSize; i++ {
		mockBatch.Slots = append(mockBatch.Slots,
			&mixmessages.Slot{
				MessagePayload: []byte{byte(i)},
			})
	}

	mockBatch.ForPhase = int32(mockPhase.GetType())
	mockBatch.Round = &mixmessages.RoundInfo{ID: uint64(roundID)}

	//send the mockBatch to the impl
	impl.PostPhase(mockBatch)

	//check the mock phase to see if the correct result has been stored
	for index := range mockBatch.Slots {
		if mockPhase.chunks[index].Begin() != uint32(index) {
			t.Errorf("PostPhase: output chunk not equal to passed;"+
				"Expected: %v, Recieved: %v", index, mockPhase.chunks[index].Begin())
		}

		if mockPhase.indices[index] != uint32(index) {
			t.Errorf("PostPhase: output index  not equal to passed;"+
				"Expected: %v, Recieved: %v", index, mockPhase.indices[index])
		}
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

/* Mock Graph */
type mockCryptop struct{}

func (*mockCryptop) GetName() string      { return "mockCryptop" }
func (*mockCryptop) GetInputSize() uint32 { return 1 }

type mockStream struct{}

func (*mockStream) Input(uint32, *mixmessages.Slot) error { return nil }
func (*mockStream) Output(uint32) *mixmessages.Slot       { return nil }
func (*mockStream) GetName() string {
	return "mockStream"
}
func (*mockStream) Link(*cyclic.Group, uint32, ...interface{}) {}

/*Mock Phase*/
type MockPhase struct {
	graph        *services.Graph
	chunks       []services.Chunk
	indices      []uint32
	stateChecker phase.GetState
	Ptype        phase.Type
}

func (mp *MockPhase) Send(chunk services.Chunk) {
	mp.chunks = append(mp.chunks, chunk)
}

func (mp *MockPhase) Input(index uint32, slot *mixmessages.Slot) error {
	if len(slot.Salt) != 0 {
		return errors.New("error to test edge case")
	}
	mp.indices = append(mp.indices, index)
	return nil
}

func (mp *MockPhase) ConnectToRound(id id.Round, setState phase.Transition,
	getState phase.GetState) {
	mp.stateChecker = getState
	return
}

func (mp *MockPhase) GetState() phase.State     { return mp.stateChecker() }
func (mp *MockPhase) GetGraph() *services.Graph { return mp.graph }

func (*MockPhase) EnableVerification()                    { return }
func (*MockPhase) GetRoundID() id.Round                   { return 0 }
func (mp *MockPhase) GetType() phase.Type                 { return mp.Ptype }
func (*MockPhase) AttemptTransitionToQueued() bool        { return true }
func (*MockPhase) TransitionToRunning()                   { return }
func (*MockPhase) UpdateFinalStates() bool                { return false }
func (*MockPhase) GetTransmissionHandler() phase.Transmit { return nil }
func (*MockPhase) GetTimeout() time.Duration              { return 0 }
func (*MockPhase) Cmp(phase.Phase) bool                   { return false }
func (*MockPhase) String() string                         { return "" }

func initMockPhase() *MockPhase {
	gc := services.NewGraphGenerator(1, nil, uint8(runtime.NumCPU()), services.AutoOutputSize, 0)
	g := gc.NewGraph("MockGraph", &mockStream{})
	var mockModule services.Module
	mockModule.Adapt = func(stream services.Stream,
		cryptop cryptops.Cryptop, chunk services.Chunk) error {
		return nil
	}
	mockModule.Cryptop = &mockCryptop{}
	mockModuleCopy := mockModule.DeepCopy()
	g.First(mockModuleCopy)
	g.Last(mockModuleCopy)
	return &MockPhase{graph: g}
}

func initImplGroup() *cyclic.Group {
	primeString := "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
		"29024E088A67CC74020BBEA63B139B22514A08798E3404DD" +
		"EF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245" +
		"E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7ED" +
		"EE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3D" +
		"C2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F" +
		"83655D23DCA3AD961C62F356208552BB9ED529077096966D" +
		"670C354E4ABC9804F1746C08CA18217C32905E462E36CE3B" +
		"E39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9" +
		"DE2BCBF6955817183995497CEA956AE515D2261898FA0510" +
		"15728E5A8AACAA68FFFFFFFFFFFFFFFF"
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16), large.NewInt(2), large.NewInt(1283))
	return grp
}

func buildMockTopology(numNodes int) *circuit.Circuit {
	var nodeIDs []*id.Node

	//Build IDs
	for i := 0; i < numNodes; i++ {
		nodIDBytes := make([]byte, id.NodeIdLen)
		nodIDBytes[0] = byte(i + 1)
		nodeID := id.NewNodeFromBytes(nodIDBytes)
		nodeIDs = append(nodeIDs, nodeID)
	}

	//Build the topology
	return circuit.New(nodeIDs)
}

func TestPostRoundPublicKeyFunc(t *testing.T) {

	grp := initImplGroup()
	topology := buildMockTopology(5)

	params := conf.Params{
		Groups: conf.Groups{CMix: grp},
		NodeID: topology.GetNodeAtIndex(1),
	}

	instance := server.CreateServerInstance(params, &globals.UserMap{})

	batchSize := uint32(11)
	roundID := id.Round(0)

	mockPhase := initMockPhase()
	mockPhase.Ptype = phase.PrecompShare

	tagKey := mockPhase.GetType().String() + "Verification"
	responseMap := make(phase.ResponseMap)
	responseMap[tagKey] = phase.NewResponse(
		phase.ResponseDefinition{
			PhaseAtSource:  mockPhase.GetType(),
			ExpectedStates: []phase.State{phase.Available},
			PhaseToExecute: mockPhase.GetType()},
	)

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

	params := conf.Params{
		Groups: conf.Groups{CMix: grp},
		NodeID: topology.GetNodeAtIndex(0),
	}

	instance := server.CreateServerInstance(params, &globals.UserMap{})

	batchSize := uint32(11)
	roundID := id.Round(0)

	mockPhase := initMockPhase()
	mockPhase.Ptype = phase.PrecompShare

	tagKey := mockPhase.GetType().String() + "Verification"
	responseMap := make(phase.ResponseMap)
	responseMap[tagKey] = phase.NewResponse(
		phase.ResponseDefinition{
			PhaseAtSource:  mockPhase.GetType(),
			ExpectedStates: []phase.State{phase.Available},
			PhaseToExecute: mockPhase.GetType()},
	)

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

	params := conf.Params{
		Groups: conf.Groups{CMix: grp},
		NodeID: topology.GetNodeAtIndex(0),
	}

	instance := server.CreateServerInstance(params, &globals.UserMap{})

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

	params := conf.Params{
		Groups: conf.Groups{CMix: grp},
		NodeID: topology.GetNodeAtIndex(0),
	}
	instance := server.CreateServerInstance(params, &globals.UserMap{})

	roundID := id.Round(45)
	// Is this the right setup for the response?
	response := phase.NewResponse(
		phase.ResponseDefinition{
			PhaseAtSource:  phase.PrecompReveal,
			ExpectedStates: []phase.State{phase.Available},
			PhaseToExecute: phase.PrecompReveal},
	)
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

		params := conf.Params{
			Groups: conf.Groups{CMix: grp},
			NodeID: topology.GetNodeAtIndex(i),
		}

		instances = append(instances, server.CreateServerInstance(
			params, &globals.UserMap{}))
	}
	instances[0].InitFirstNode()

	// Set up a round on all the instances
	roundID := id.Round(45)
	for i := 0; i < numNodes; i++ {
		response := phase.NewResponse(phase.ResponseDefinition{
			PhaseAtSource:  phase.PrecompReveal,
			ExpectedStates: []phase.State{phase.Available},
			PhaseToExecute: phase.PrecompReveal})

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

func TestReceiveFinishRealtime(t *testing.T) {
	// Smoke tests the management part of PostPrecompResult
	grp := initImplGroup()
	const numNodes = 5
	topology := buildMockTopology(numNodes)

	// Set instance for first node
	params := conf.Params{
		Groups: conf.Groups{CMix: grp},
		NodeID: topology.GetNodeAtIndex(0),
	}
	instance := server.CreateServerInstance(params, &globals.UserMap{})
	instance.InitFirstNode()

	// Set up a round first node
	roundID := id.Round(45)

	response := phase.NewResponse(phase.ResponseDefinition{
		PhaseAtSource:  phase.RealPermute,
		ExpectedStates: []phase.State{phase.Available},
		PhaseToExecute: phase.RealPermute})

	responseMap := make(phase.ResponseMap)
	responseMap["Completed"] = response

	p := initMockPhase()
	p.Ptype = phase.RealPermute

	rnd := round.New(grp, &globals.UserMap{}, roundID, []phase.Phase{p},
		responseMap, topology, topology.GetNodeAtIndex(0), 3)

	instance.GetRoundManager().AddRound(rnd)

	// Initially, there should be zero rounds on the precomp queue
	if len(instance.GetFinishedRounds()) != 0 {
		t.Error("Expected completed precomps to be empty")
	}

	var err error

	info := mixmessages.RoundInfo{
		ID: uint64(roundID),
	}

	go func() {
		err = ReceiveFinishRealtime(instance, &info)
	}()

	var finishedRoundID id.Round

	select {
	case finishedRoundID = <-instance.GetFinishedRounds():
	case <-time.After(2 * time.Second):
	}

	if err != nil {
		t.Errorf("ReceiveFinishRealtime: errored: %+v", err)
	}

	if finishedRoundID != roundID {
		t.Errorf("ReceiveFinishRealtime: Expected round %v to finish, "+
			"recieved %v", roundID, finishedRoundID)
	}
}
