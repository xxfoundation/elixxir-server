package receivers

import (
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/node"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/testUtil"
	"testing"
)

func TestNewImplementation_PostPhase(t *testing.T) {
	batchSize := uint32(11)
	roundID := id.Round(0)
	grp := initImplGroup()

	def := server.Definition{
		CmixGroup:       grp,
		Topology:        connect.NewCircuit(buildMockNodeIDs(2)),
		UserRegistry:    &globals.UserMap{},
		ResourceMonitor: &measure.ResourceMonitor{},
	}

	def.ID = def.Topology.GetNodeAtIndex(0)

	instance, _ := server.CreateServerInstance(&def, NewImplementation, false)

	mockPhase := testUtil.InitMockPhase(t)

	responseMap := make(phase.ResponseMap)
	responseMap[mockPhase.GetType().String()] =
		phase.NewResponse(phase.ResponseDefinition{mockPhase.GetType(),
			[]phase.State{phase.Active}, mockPhase.GetType()})

	topology := instance.GetTopology()

	r := round.New(grp, &globals.UserMap{}, roundID, []phase.Phase{mockPhase},
		responseMap, topology, topology.GetNodeAtIndex(0), batchSize,
		instance.GetRngStreamGen(), "0.0.0.0")

	instance.GetRoundManager().AddRound(r)

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
	lastNodeId := topology.GetNodeAtIndex(lastNodeIndex).String()
	fakeHost, err := connect.NewHost(lastNodeId, "", nil, true, true)
	if err != nil {
		t.Errorf("Failed to create fakeHost, %s", err)
	}

	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          fakeHost,
	}
	// send the mockBatch to the impl
	impl.PostPhase(mockBatch, auth)

	//check the mock phase to see if the correct result has been stored
	for index := range mockBatch.Slots {
		if mockPhase.GetChunks()[index].Begin() != uint32(index) {
			t.Errorf("PostPhase: output chunk not equal to passed;"+
				"Expected: %v, Recieved: %v", index, mockPhase.GetChunks()[index].Begin())
		}

		if mockPhase.GetIndices()[index] != uint32(index) {
			t.Errorf("PostPhase: output index  not equal to passed;"+
				"Expected: %v, Recieved: %v", index, mockPhase.GetIndices()[index])
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

	instance := mockServerInstance(t)
	mockPhase := testUtil.InitMockPhase(t)

	topology := instance.GetTopology()

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
	lastNodeId := topology.GetNodeAtIndex(lastNodeIndex).String()
	fakeHost, err := connect.NewHost(lastNodeId, "", nil, true, true)
	if err != nil {
		t.Errorf("Failed to create fakeHost, %s", err)
	}

	auth := &connect.Auth{
		IsAuthenticated: false,
		Sender:          fakeHost,
	}

	ReceivePostPhase(mockBatch, instance, auth)

	t.Errorf("Expected error case, should not be able to ReceivePostPhase when not authenticated")

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

	instance := mockServerInstance(t)
	mockPhase := testUtil.InitMockPhase(t)

	topology := instance.GetTopology()

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
	lastNodeId := topology.GetNodeAtIndex(lastNodeIndex).String()
	fakeHost, err := connect.NewHost(lastNodeId, "", nil, true, true)
	if err != nil {
		t.Errorf("Failed to create fakeHost, %s", err)
	}

	auth := &connect.Auth{
		IsAuthenticated: false,
		Sender:          fakeHost,
	}

	ReceivePostPhase(mockBatch, instance, auth)

	t.Errorf("Expected error case, should not be able to ReceivePostPhase when not authenticated")

}

// Error Path
func TestStreamPostPhase_NoAuth(t *testing.T) {
	batchSize := uint32(11)
	roundID := id.Round(0)

	instance := mockServerInstance(t)
	mockPhase := testUtil.InitMockPhase(t)

	topology := instance.GetTopology()

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
	lastNodeId := topology.GetNodeAtIndex(lastNodeIndex).String()
	fakeHost, err := connect.NewHost(lastNodeId, "", nil, true, true)
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

	instance := mockServerInstance(t)
	mockPhase := testUtil.InitMockPhase(t)

	topology := instance.GetTopology()

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
	lastNodeId := topology.GetNodeAtIndex(lastNodeIndex).String()
	fakeHost, err := connect.NewHost(lastNodeId, "", nil, true, true)
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

	grp := initImplGroup()

	def := server.Definition{
		CmixGroup:       grp,
		Topology:        connect.NewCircuit(buildMockNodeIDs(2)),
		UserRegistry:    &globals.UserMap{},
		ResourceMonitor: &measure.ResourceMonitor{},
	}
	def.ID = def.Topology.GetNodeAtIndex(0)

	instance, _ := server.CreateServerInstance(&def, NewImplementation, false)
	mockPhase := testUtil.InitMockPhase(t)

	responseMap := make(phase.ResponseMap)
	responseMap[mockPhase.GetType().String()] =
		phase.NewResponse(phase.ResponseDefinition{mockPhase.GetType(),
			[]phase.State{phase.Active}, mockPhase.GetType()})

	topology := instance.GetTopology()

	r := round.New(grp, &globals.UserMap{}, roundID, []phase.Phase{mockPhase},
		responseMap, topology, topology.GetNodeAtIndex(0), batchSize,
		instance.GetRngStreamGen(), "0.0.0.0")

	instance.GetRoundManager().AddRound(r)

	// get the impl
	impl := node.NewImplementation(instance)

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
	lastNodeId := topology.GetNodeAtIndex(lastNodeIndex).String()
	fakeHost, err := connect.NewHost(lastNodeId, "", nil, true, true)
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
				"Expected: %v, Recieved: %v", index, mockPhase.GetChunks()[index].Begin())
		}

		if mockPhase.GetIndices()[index] != uint32(index) {
			t.Errorf("StreamPostPhase: output index  not equal to passed;"+
				"Expected: %v, Recieved: %v", index, mockPhase.GetIndices()[index])
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
