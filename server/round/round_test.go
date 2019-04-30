package round

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
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
	var phases []*phase.Phase
	phases = append(phases, phase.New(initMockGraph(services.
		NewGraphGenerator(1, nil, 1, 1, 1)),
		phase.REAL_PERMUTE, func(phase *phase.Phase,
			nal *services.NodeAddressList, getSlot phase.GetChunk,
			getMessage phase.GetMessage) {
			return
		}, time.Minute))
	myLoc := 1
	// Node address list is used to test node addresses and myLoc
	nodeAddressList := services.NewNodeAddressList(
		[]services.NodeAddress{{
			Cert:    "not a cert",
			Address: "127.0.0.1",
			Id:      0,
		}}, myLoc)

	round := New(grp, roundId, phases, nodeAddressList.GetAllNodesAddress(),
		myLoc, 5)

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
		t.Error("Phase list differed")
	}
	// Covers node address list and myLoc
	if !reflect.DeepEqual(round.GetNodeAddressList(), nodeAddressList) {
		t.Error("Node address list differed")
	}

	// Because it's a lot of rigamarole to create the round again,
	// here's coverage for GetPhase and GetCurrentPhase
	// should return nil
	nilPhase := round.GetPhase(phase.PRECOMP_GENERATION)
	if nilPhase != nil {
		t.Fatal("Should have gotten a nil phase")
	}
	actualPhase := round.GetPhase(phase.REAL_PERMUTE)
	if !actualPhase.Cmp(phases[0]) {
        t.Error("Phases differed")
	}
	actualPhaseType := round.GetCurrentPhase().GetType()
	if actualPhaseType != phase.REAL_PERMUTE {
		t.Error("Current phase should have been realtime permute")
	}
}
