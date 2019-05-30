package round

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/services"
	"reflect"
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

	handler := func(network *node.NodeComms, batchSize uint32,
		roundId id.Round, phaseTy phase.Type, getSlot phase.GetChunk,
		getMessage phase.GetMessage, nodes *circuit.Circuit, nid *id.Node) error {
		return nil
	}

	phases = append(phases, phase.New(phase.Definition{initMockGraph(services.
		NewGraphGenerator(1, nil, 1,
			1, 1)),
		phase.RealPermute, handler, time.Minute,
		false}))

	topology := circuit.New([]*id.Node{&id.Node{}})

	round := New(grp, roundId, phases, nil, topology,
		&id.Node{}, 5)

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
	if round.GetCurrentPhase().GetState() != phase.Available {
		t.Errorf("phase's state is %v, should have been Available",
			round.GetCurrentPhase().GetState())
	}
	// This should fail...
	panicChan := make(chan bool)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				panicChan <- false
				return
			}
			t.Errorf("Round: should be unreachable code")
		}()
		round.GetCurrentPhase().TransitionToRunning()
		panicChan <- true
	}()

	runningSuccess := <-panicChan
	if runningSuccess {
		t.Error("Shouldn't have been able to successfully increment phase to" +
			" Queued")
	}
	// and the state should remain Initialized
	if round.GetCurrentPhase().GetState() != phase.Available {
		t.Error("phase's state should have remained Available")
	}

	// However, setting the state to Queued should succeed
	if !round.GetCurrentPhase().AttemptTransitionToQueued() {
		t.Error("Should have been able to take state from Available to Queued")
	}
	// And, the state should be set to Queued
	if round.GetCurrentPhase().GetState() != phase.Queued {
		t.Error("phase's state should be Queued")
	}
}
