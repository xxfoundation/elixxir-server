///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package internal

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/internal/state"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/elixxir/server/testUtil"
	"gitlab.com/xx_network/comms/connect"
	"runtime"
	"sync"
	"testing"
	"time"
)

var pString = "9DB6FB5951B66BB6FE1E140F1D2CE5502374161FD6538DF1648218642F0B5C48" +
	"C8F7A41AADFA187324B87674FA1822B00F1ECF8136943D7C55757264E5A1A44F" +
	"FE012E9936E00C1D3E9310B01C7D179805D3058B2A9F4BB6F9716BFE6117C6B5" +
	"B3CC4D9BE341104AD4A80AD6C94E005F4B993E14F091EB51743BF33050C38DE2" +
	"35567E1B34C3D6A5C0CEAA1A0F368213C3D19843D0B4B09DCB9FC72D39C8DE41" +
	"F1BF14D4BB4563CA28371621CAD3324B6A2D392145BEBFAC748805236F5CA2FE" +
	"92B871CD8F9C36D3292B5509CA8CAA77A2ADFC7BFD77DDA6F71125A7456FEA15" +
	"3E433256A2261C6A06ED3693797E7995FAD5AABBCFBE3EDA2741E375404AE25B"

var gString = "5C7FF6B06F8F143FE8288433493E4769C4D988ACE5BE25A0E24809670716C613" +
	"D7B0CEE6932F8FAA7C44D2CB24523DA53FBE4F6EC3595892D1AA58C4328A06C4" +
	"6A15662E7EAA703A1DECF8BBB2D05DBE2EB956C142A338661D10461C0D135472" +
	"085057F3494309FFA73C611F78B32ADBB5740C361C9F35BE90997DB2014E2EF5" +
	"AA61782F52ABEB8BD6432C4DD097BC5423B285DAFB60DC364E8161F4A2A35ACA" +
	"3A10B1C4D203CC76A470A33AFDCBDD92959859ABD8B56E1725252D78EAC66E71" +
	"BA9AE3F1DD2487199874393CD4D832186800654760E1E34C09E4D155179F9EC0" +
	"DC4473F996BDCE6EED1CABED8B6F116F7AD9CF505DF0F998E34AB27514B0FFE7"

var pPrime = large.NewIntFromString(pString, 16)
var g = large.NewIntFromString(gString, 16)

var grp = cyclic.NewGroup(pPrime, g)

// This MockPhase is only used to test denoting phase completion while the queue
// runner isn't running
type MockPhase struct {
	chunks  []services.Chunk
	indices []uint32
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

func (*MockPhase) EnableVerification() { return }
func (*MockPhase) ConnectToRound(id id.Round, setState phase.Transition,
	getState phase.GetState) {
	return
}

func (*MockPhase) GetGraph() *services.Graph                    { return nil }
func (*MockPhase) GetRoundID() id.Round                         { return 0 }
func (*MockPhase) GetType() phase.Type                          { return 0 }
func (*MockPhase) GetState() phase.State                        { return 0 }
func (*MockPhase) TransitionToRunning()                         { return }
func (*MockPhase) AttemptToQueue(queue chan<- phase.Phase) bool { return false }
func (*MockPhase) IsQueued() bool                               { return false }
func (*MockPhase) UpdateFinalStates()                           { return }
func (*MockPhase) GetTransmissionHandler() phase.Transmit       { return nil }
func (*MockPhase) GetTimeout() time.Duration                    { return 5 * time.Second }
func (*MockPhase) Cmp(phase.Phase) bool                         { return false }
func (*MockPhase) String() string                               { return "" }
func (*MockPhase) Measure(string)                               { return }
func (*MockPhase) GetMeasure() measure.Metrics                  { return *new(measure.Metrics) }

func TestResourceQueue_DenotePhaseCompletion(t *testing.T) {
	q := initQueue()
	p := &MockPhase{}
	q.GetPhaseQueue() <- p
	q.DenotePhaseCompletion(p)
	// After these calls, the finishing queue should have something on it
	if len(q.finishChan) != 1 {
		t.Error("There should be a phase in the channel of finished phases")
	}
}

func TestResourceQueue_RunOne(t *testing.T) {
	impl := func(*Instance) *node.Implementation {
		return node.NewImplementation()
	}

	// In this case, we actually need to set up and run the queue runner
	q := initQueue()
	nid := GenerateId(t)

	topology := connect.NewCircuit([]*id.ID{nid})
	def := Definition{
		ID:              nid,
		UserRegistry:    &globals.UserMap{},
		ResourceMonitor: &measure.ResourceMonitor{},
		FullNDF:         testUtil.NDF,
		PartialNDF:      testUtil.NDF,
		Flags:           Flags{DisableIpOverride: true},
	}
	def.Gateway.ID = nid.DeepCopy()
	def.Gateway.ID.SetType(id.Gateway)
	m := state.NewMachine(dummyStates)
	instance, _ := CreateServerInstance(&def, impl, m, "1.1.0")
	roundID := id.Round(1)
	p := makeTestPhase(instance, phase.PrecompGeneration, roundID)
	// Then, we need a response map for the phase
	responseMap := make(phase.ResponseMap)
	// Is this the correct key for the map?
	responseMap[phase.PrecompGeneration.String()] =
		phase.NewResponse(phase.ResponseDefinition{
			phase.PrecompGeneration,
			[]phase.State{phase.Active},
			phase.PrecompGeneration,
		})

	myGrp := cyclic.NewGroup(pPrime, g)

	r, err := round.New(myGrp, instance.GetUserRegistry(), roundID, []phase.Phase{p},
		responseMap, topology, instance.GetID(), 1,
		instance.GetRngStreamGen(), nil, "0.0.0.0", nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}
	instance.GetRoundManager().AddRound(r)

	if p.GetState() != phase.Active {
		t.Error("Before enqueueing, the phase's state should be Active")
	}

	p.AttemptToQueue(q.GetPhaseQueue())
	// Verify state before the queue runs
	if len(q.phaseQueue) != 1 {
		t.Error("Before running, the queue should have one phase")
	}
	if !p.IsQueued() {
		t.Error("After enqueueing, the phase's state should be Queued")
	}
	go q.run(instance)
	time.Sleep(100 * time.Millisecond)
	// Verify state while the queue is running
	iWasCalledLck.Lock()
	if !iWasCalled {
		t.Error("Transmission handler never got called")
	}
	iWasCalledLck.Unlock()
	//log := io.Writer()
	//log
	if len(q.phaseQueue) != 0 {
		t.Error("The phase queue should have been emptied after the queue ran" +
			" the only phase")
	}
	q.DenotePhaseCompletion(p)
	time.Sleep(20 * time.Millisecond)
}

type mockStream struct{}

func (*mockStream) Input(uint32, *mixmessages.Slot) error { return nil }
func (*mockStream) Output(uint32) *mixmessages.Slot       { return nil }
func (*mockStream) GetName() string {
	return "mockStream"
}
func (*mockStream) Link(*cyclic.Group, uint32, ...interface{}) {}

type mockCryptop struct{}

func (*mockCryptop) GetName() string {
	return "mockCryptop"
}
func (*mockCryptop) GetInputSize() uint32 {
	return services.AutoInputSize
}

var iWasCalled bool
var iWasCalledLck sync.Mutex

func makeTestPhase(instance *Instance, name phase.Type,
	roundID id.Round) phase.Phase {

	// FIXME We need to be able to killChan this,
	//  or tell whether something was killed before calling DenotePhaseComplete.
	//  It could be done by changing the way that GetChunk works/the GetChunk
	//  header.
	transmissionHandler := func(roundID id.Round, instance phase.GenericInstance, getChunk phase.GetChunk,
		getMessage phase.GetMessage) error {
		iWasCalledLck.Lock()
		defer iWasCalledLck.Unlock()
		iWasCalled = true
		return nil
	}
	timeout := 500 * time.Millisecond
	p := phase.New(phase.Definition{makeTestGraph(instance, 1), name, transmissionHandler,
		timeout, false})
	return p
}

func makeTestGraph(instance *Instance, batchSize uint32) *services.Graph {
	graphGen := services.NewGraphGenerator(4,
		uint8(runtime.NumCPU()), 1, 1)
	graph := graphGen.NewGraph("TestGraph", &mockStream{})

	mockModule := services.Module{
		Adapt: func(stream services.Stream, cryptop cryptops.Cryptop, chunk services.Chunk) error {
			return nil
		},
		Cryptop: &mockCryptop{},
		// Why wasn't I able to get AutoInputSize working?
		// Is it supposed to be used here?
		InputSize:      4,
		StartThreshold: 0,
		Name:           "mockModule",
		NumThreads:     services.AutoNumThreads,
	}
	firstGraphNode := mockModule.DeepCopy()
	secondGraphNode := mockModule.DeepCopy()
	graph.First(firstGraphNode)
	graph.Connect(firstGraphNode, secondGraphNode)
	graph.Last(secondGraphNode)

	return graph
}
