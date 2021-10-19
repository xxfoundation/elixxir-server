///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package round

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/gpumathsgo/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"runtime"
	"testing"
	"time"
)

type mockCryptop struct{}

func (*mockCryptop) GetName() string      { return "mockCryptop" }
func (*mockCryptop) GetInputSize() uint32 { return 1 }

type mockStream struct{}

func (*mockStream) Input(uint32, *mixmessages.Slot) error      { return nil }
func (*mockStream) Output(uint32) *mixmessages.Slot            { return nil }
func (*mockStream) GetName() string                            { return "mockStream" }
func (*mockStream) Link(*cyclic.Group, uint32, ...interface{}) {}

// We can't use real graphs from realtime or precomputation phases, because
// they import Round and that causes an import cycle.
// This is a valid graph with one module marked fist and last that does nothing
func initMockGraph(gg services.GraphGenerator) *services.Graph {
	graph := gg.NewGraph("MockGraph", &mockStream{})
	var mockModule services.Module
	mockModule.Adapt = func(stream services.Stream, cryptop cryptops.Cryptop, chunk services.Chunk) error {
		return nil
	}
	mockModule.Cryptop = &mockCryptop{}
	mockModuleCopy := mockModule.DeepCopy()
	graph.First(mockModuleCopy)
	graph.Last(mockModuleCopy)
	return graph
}

func TestNew(t *testing.T) {
	// After calling New() on a round,
	// the round should be fully initialized and ready for use
	roundId := id.Round(58)
	var phases []phase.Phase

	handler := func(roundID id.Round, instance phase.GenericInstance, getChunk phase.GetChunk, getMessage phase.GetMessage) error {
		return nil
	}

	phases = append(phases, phase.New(phase.Definition{Graph: initMockGraph(services.
		NewGraphGenerator(1, 1,
			1, 1)),
		Type: phase.RealPermute, TransmissionHandler: handler, Timeout: time.Minute}))

	topology := connect.NewCircuit([]*id.ID{{}})

	round, err := New(grp, roundId, phases, nil, topology, &id.ID{}, 5, fastRNG.NewStreamGenerator(10000,
		uint(runtime.NumCPU()), csprng.NewSystemRNG), nil, "0.0.0.0", nil, nil, nil, nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}

	if round.GetID() != roundId {
		t.Error("Round ID wasn't set correctly")
	}
	// The round's buffer should use the same group fingerprint as the passed
	// group
	if round.GetBuffer().CypherPublicKey.GetGroupFingerprint() != grp.GetFingerprint() {
		t.Error("Round's group was different")
	}
	// The phase slice should be aliased given the way the constructor works,
	// so any mutations New makes to the phase list should be reflected in the
	// original copy
	if !reflect.DeepEqual(round.phases, phases) {
		t.Error("phase list differed")
	}
	// Covers node address list and myLoc
	if !reflect.DeepEqual(round.GetTopology(), topology) {
		t.Error("Node address list differed")
	}

	// Because it's a lot of rigamarole to create the round again,
	// here's coverage for GetPhase and GetCurrentPhase
	// should return nil
	nilPhase, _ := round.GetPhase(phase.PrecompGeneration)
	if nilPhase != nil {
		t.Fatal("Should have gotten a nil phase")
	}
	actualPhase, _ := round.GetPhase(phase.RealPermute)
	if !actualPhase.Cmp(phases[0]) {
		t.Error("Phases differed")
	}
	actualPhaseType := round.GetCurrentPhase().GetType()
	if actualPhaseType != phase.RealPermute {
		t.Error("Current phase should have been realtime permute")
	}
	// Try getting and setting the state of the phase
	if round.GetCurrentPhase().GetState() != phase.Active {
		t.Errorf("phase's state is %v, should have been Available",
			round.GetCurrentPhase().GetState())
	}
}

func TestRound_GetMeasurements(t *testing.T) {
	// After calling New() on a round, the round should be fully initialized and
	// ready for use
	roundId := id.Round(58)
	var phases []phase.Phase

	handler := func(roundID id.Round, instance phase.GenericInstance, getChunk phase.GetChunk, getMessage phase.GetMessage) error {
		return nil
	}

	newGraph := services.NewGraphGenerator(1, 1, 1, 1)

	newPhaseDef := phase.Definition{
		Graph:               initMockGraph(newGraph),
		Type:                phase.RealPermute,
		TransmissionHandler: handler,
		Timeout:             time.Minute,
	}

	phases = append(phases, phase.New(newPhaseDef))

	nidStr := id.NewIdFromString("123", id.Node, t)
	nid := id.NewIdFromUInt(uint64(123), id.Node, t)
	topology := connect.NewCircuit([]*id.ID{nid})

	round, err := New(grp, roundId, phases, nil, topology, nid, 5, fastRNG.NewStreamGenerator(10000,
		uint(runtime.NumCPU()), csprng.NewSystemRNG), nil, "0.0.0.0", nil, nil, nil, nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}

	timeNow := time.Now()
	resourceMetric := measure.ResourceMetric{
		Time:          timeNow,
		MemAllocBytes: 10,
		NumThreads:    100,
	}
	resourceMonitor := measure.ResourceMonitor{}
	resourceMonitor.Set(resourceMetric)
	numNodes := 1
	index := 0
	roundMetrics := round.GetMeasurements(nidStr, numNodes, index,
		resourceMonitor.Get())

	if !roundMetrics.NodeID.Cmp(nidStr) {
		t.Errorf("Round metrics has incorrect node id expected %v got %v",
			nidStr, roundMetrics.NodeID)
	}
	if roundMetrics.Index != index {
		t.Errorf("Round metrics has incorrect index expected %v got %v",
			index, roundMetrics.Index)
	}
	if roundMetrics.RoundID != 58 {
		t.Errorf("Round metrics has incorrect round id expected %v got %v",
			58, roundMetrics.RoundID)
	}
	if !reflect.DeepEqual(resourceMetric, roundMetrics.ResourceMetric) {
		t.Errorf("Round metrics has mismatching resource metricsexpected %v got %v",
			resourceMetric, roundMetrics.ResourceMetric)
	}
}

func TestRound_StartRoundTrip(t *testing.T) {
	var phases []phase.Phase
	roundId := id.Round(58)
	phases = append(phases, phase.New(phase.Definition{Graph: initMockGraph(services.
		NewGraphGenerator(1, 1,
			1, 1)),
		Type: phase.RealPermute, TransmissionHandler: nil, Timeout: time.Minute}))

	topology := connect.NewCircuit([]*id.ID{{}})

	round, err := New(grp, roundId, phases, nil, topology, &id.ID{}, 5, fastRNG.NewStreamGenerator(10000,
		uint(runtime.NumCPU()), csprng.NewSystemRNG), nil, "0.0.0.0", nil, nil, nil, nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}
	payload := "NULL/ACK"
	unsetStart := round.rtStartTime
	round.StartRoundTrip(payload)

	if payload != round.GetRTPayload() {
		t.Errorf("StartRoundTrip did not set rtPayload\n\texpected: %s\n\tfound: %s", payload, round.GetRTPayload())
	} else if !round.rtStarted {
		t.Error("StartRoundTrip did not set rtStarted")
	} else if round.rtStartTime == unsetStart {
		t.Error("StartRoundTrip did not set start time")
	} else if round.GetRTPayload() != payload {
		t.Error("StartRoundTrip did not set payload")
	}
}

func TestRound_StopRoundTrip(t *testing.T) {
	var phases []phase.Phase
	roundId := id.Round(58)
	phases = append(phases, phase.New(phase.Definition{Graph: initMockGraph(services.
		NewGraphGenerator(1, 1,
			1, 1)),
		Type: phase.RealPermute, TransmissionHandler: nil, Timeout: time.Minute}))

	topology := connect.NewCircuit([]*id.ID{{}})

	round, err := New(grp, roundId, phases, nil, topology, &id.ID{}, 5, fastRNG.NewStreamGenerator(10000,
		uint(runtime.NumCPU()), csprng.NewSystemRNG), nil, "0.0.0.0", nil, nil, nil, nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}
	unsetStop := round.rtEndTime

	err = round.StopRoundTrip()
	if err == nil {
		t.Errorf("StopRoundTrip should error if rtStarted not set: %+v", err)
	}

	round.rtStarted = true
	err = round.StopRoundTrip()
	if err != nil {
		t.Errorf("StopRoundTrip should not error if rtStarted is set: %+v", err)
	}
	if round.GetRTEnd() == unsetStop {
		t.Error("StopRoundTrip did not set stop time")
	} else if round.roundMetrics.RTDurationMilli == 0 {
		t.Error("StopRoundTrip did not set duration")
	}
}
