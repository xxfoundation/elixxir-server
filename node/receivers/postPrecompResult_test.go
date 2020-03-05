package receivers

import (
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/server/state"
	"gitlab.com/elixxir/server/testUtil"
	"testing"
)

// Shows that ReceivePostPrecompResult panics when the round isn't in
// the round manager
func TestPostPrecompResultFunc_Error_NoRound(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("There was no panic when an invalid round was passed")
		}
	}()
	//grp := initImplGroup()
	topology := connect.NewCircuit(buildMockNodeIDs(5))
	def := server.Definition{
		UserRegistry:    &globals.UserMap{},
		ResourceMonitor: &measure.ResourceMonitor{},
	}
	def.ID = topology.GetNodeAtIndex(0)

	instance, _ := server.CreateServerInstance(&def, NewImplementation, [current.NUM_STATES]state.Change{}, false)

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

	// We haven't set anything up,
	// so this should panic because the round can't be found
	err = ReceivePostPrecompResult(instance, 0, []*mixmessages.Slot{}, auth)

	if err == nil {
		t.Error("Didn't get an error from a nonexistent round")
	}
}

// Shows that ReceivePostPrecompResult returns an error when there are a wrong
// number of slots in the message
func TestPostPrecompResultFunc_Error_WrongNumSlots(t *testing.T) {
	// Smoke tests the management part of PostPrecompResult
	grp := initImplGroup()

	topology := connect.NewCircuit(buildMockNodeIDs(5))
	def := server.Definition{
		UserRegistry:    &globals.UserMap{},
		ResourceMonitor: &measure.ResourceMonitor{},
	}
	def.ID = topology.GetNodeAtIndex(0)

	instance, _ := server.CreateServerInstance(&def, NewImplementation, [current.NUM_STATES]state.Change{}, false)

	roundID := id.Round(45)
	// Is this the right setup for the response?
	response := phase.NewResponse(
		phase.ResponseDefinition{
			PhaseAtSource:  phase.PrecompReveal,
			ExpectedStates: []phase.State{phase.Active},
			PhaseToExecute: phase.PrecompReveal},
	)
	responseMap := make(phase.ResponseMap)
	responseMap[phase.PrecompReveal.String()+"Verification"] = response
	// This is quite a bit of setup...
	p := testUtil.InitMockPhase(t)
	p.Ptype = phase.PrecompReveal
	instance.GetRoundManager().AddRound(round.New(grp,
		instance.GetUserRegistry(), roundID, []phase.Phase{p}, responseMap,
		topology, topology.GetNodeAtIndex(0), 3,
		instance.GetRngStreamGen(), "0.0.0.0"))

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

	// This should give an error because we give it fewer slots than are in the
	// batch
	err = ReceivePostPrecompResult(instance, uint64(roundID), []*mixmessages.Slot{}, auth)

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
	nodeIDs := buildMockNodeIDs(5)

	// Set up all the instances
	var instances []*server.Instance
	for i := 0; i < numNodes; i++ {
		topology := connect.NewCircuit(nodeIDs)
		def := server.Definition{
			UserRegistry:    &globals.UserMap{},
			ResourceMonitor: &measure.ResourceMonitor{},
		}
		def.ID = topology.GetNodeAtIndex(i)
		instance, _ := server.CreateServerInstance(&def, NewImplementation, [current.NUM_STATES]state.Change{}, false)
		instances = append(instances, instance)
	}

	topology := instances[0].GetTopology()

	// Set up a round on all the instances
	roundID := id.Round(45)
	for i := 0; i < numNodes; i++ {
		response := phase.NewResponse(phase.ResponseDefinition{
			PhaseAtSource:  phase.PrecompReveal,
			ExpectedStates: []phase.State{phase.Active},
			PhaseToExecute: phase.PrecompReveal})

		responseMap := make(phase.ResponseMap)
		responseMap[phase.PrecompReveal.String()+"Verification"] = response
		// This is quite a bit of setup...
		p := testUtil.InitMockPhase(t)
		p.Ptype = phase.PrecompReveal
		instances[i].GetRoundManager().AddRound(round.New(grp,
			instances[i].GetUserRegistry(), roundID,
			[]phase.Phase{p}, responseMap, topology, topology.GetNodeAtIndex(i),
			3, instances[i].GetRngStreamGen(), "0.0.0.0"))
	}

	// Initially, there should be zero rounds on the precomp queue
	//if len(instances[0].GetCompletedPrecomps().CompletedPrecomputations) != 0 {
	//	t.Error("Expected completed precomps to be empty")
	//}

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

	// Since we give this 3 slots with the correct fields populated,
	// it should work without errors on all nodes
	for i := 0; i < numNodes; i++ {
		err := ReceivePostPrecompResult(instances[i], uint64(roundID),
			[]*mixmessages.Slot{{
				PartialPayloadACypherText: grp.NewInt(3).Bytes(),
				PartialPayloadBCypherText: grp.NewInt(4).Bytes(),
			}, {
				PartialPayloadACypherText: grp.NewInt(3).Bytes(),
				PartialPayloadBCypherText: grp.NewInt(4).Bytes(),
			}, {
				PartialPayloadACypherText: grp.NewInt(3).Bytes(),
				PartialPayloadBCypherText: grp.NewInt(4).Bytes(),
			}}, auth)

		if err != nil {
			t.Errorf("Error posting precomp on node %v: %v", i, err)
		}
	}

	// Then, after the reception handler ran successfully,
	// there should be 1 precomputation in the buffer on the first node
	// The others don't have this variable initialized
	//if len(instances[0].GetCompletedPrecomps().CompletedPrecomputations) != 1 {
	//	t.Error("Expected completed precomps to have the one precomp we posted")
	//}
}

// Tests that ReceivePostPrecompResult() returns an error when isAuthenticated
// is set to false in the Auth object.
func TestReceivePostPrecompResult_NoAuth(t *testing.T) {
	instance := mockServerInstance(t)

	fakeHost, err := connect.NewHost(instance.GetTopology().GetNodeAtIndex(0).String(), "", nil, true, true)
	if err != nil {
		t.Errorf("Failed to create fakeHost, %s", err)
	}
	auth := connect.Auth{
		IsAuthenticated: false,
		Sender:          fakeHost,
	}
	err = ReceivePostPrecompResult(instance, 0, []*mixmessages.Slot{}, &auth)

	if err == nil {
		t.Errorf("ReceivePostPrecompResult: did not error with IsAuthenticated false")
	}
}

// Tests that ReceivePostPrecompResult() returns an error when Sender is set to
// the wrong sender in the Auth object.
func TestPostPrecompResult_WrongSender(t *testing.T) {
	instance := mockServerInstance(t)

	fakeHost, err := connect.NewHost("bad", "", nil, true, true)
	if err != nil {
		t.Errorf("Failed to create fakeHost, %s", err)
	}
	auth := connect.Auth{
		IsAuthenticated: true,
		Sender:          fakeHost,
	}
	err = ReceivePostPrecompResult(instance, 0, []*mixmessages.Slot{}, &auth)

	if err == nil {
		t.Errorf("ReceivePostPrecompResult: did not error with wrong host")
	}
}
