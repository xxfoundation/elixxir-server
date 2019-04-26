package phase

import (
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/services"
	"testing"
	"time"
)

// GETTER TESTS
// Proves that Phase gets a fingerprint that represents it
func TestPhase_GetFingerprint(t *testing.T) {
	phaseType := uint32(2)
	roundID := id.Round(258)
	p := Phase{
		roundID: roundID,
		tYpe:    Type(phaseType),
	}
	fingerprint := p.GetFingerprint()
	if fingerprint.round != roundID {
		t.Error("Fingerprint round ID didn't match")
	}
	if fingerprint.tYpe != Type(phaseType) {
		t.Error("Fingerprint phase type didn't match")
	}
}

func TestPhase_GetGraph(t *testing.T) {
	g := services.Graph{}
	p := Phase{
		graph: &g,
	}
	if p.GetGraph() != &g {
		t.Error("Phase graphs were different")
	}
}

func TestPhase_GetRoundID(t *testing.T) {
	r := id.Round(562359865894179)
	p := Phase{
		roundID: r,
	}
	if p.GetRoundID() != r {
		t.Error("Round ID was different")
	}
}

func TestPhase_GetTimeout(t *testing.T) {
	timeout := 580 * time.Second
	p := Phase{
		timeout: timeout,
	}
	if p.GetTimeout() != timeout {
		t.Error("Timeout was different")
	}
}

func TestPhase_GetTransmissionHandler(t *testing.T) {
	pass := false
	handler := func(phase *Phase, nal *services.NodeAddressList,
		getSlot GetChunk, getMessage GetMessage) {
		pass = true
	}
	p := Phase{
		transmissionHandler: handler,
	}
	// This call should set pass to true
	p.GetTransmissionHandler()(nil, nil, nil, nil)
	if !pass {
		t.Error("Didn't get the correct transmission handler")
	}
}

func TestPhase_GetState(t *testing.T) {
	state := Available
	p := Phase{state: (*uint32)(&state)}
	if p.GetState() != state {
		t.Error("State was different")
	}
}

func TestPhase_GetType(t *testing.T) {
	phaseType := PRECOMP_GENERATION
	p := Phase{tYpe: phaseType}
	if p.GetType() != phaseType {
		t.Error("Type was different")
	}
}

// Other tests prove that the various fields that should be set or compared
// are set or compared correctly
func TestPhase_ReadyToReceiveData(t *testing.T) {
	state := Initialized
	p := Phase{state: (*uint32)(&state)}
	if p.ReadyToReceiveData() {
		t.Error("Initialized phase shouldn't be ready to receive")
	}
	state = Available
	if !p.ReadyToReceiveData() {
		t.Error("Available phase should be ready to receive")
	}
	state = Queued
	if !p.ReadyToReceiveData() {
		t.Error("Queued phase should be ready to receive")
	}
	state = Running
	if !p.ReadyToReceiveData() {
		t.Error("Running phase should be ready to receive")
	}
	state = Finished
	if p.ReadyToReceiveData() {
		t.Error("Finished phase should not be ready to receive")
	}
}

func TestPhase_ConnectToRound(t *testing.T) {
	g := NewStateGroup()
	var p Phase

	roundId := id.Round(55)
	p.ConnectToRound(roundId, g)

	// The Once shouldn't be allowed to run again
	pass := true
	p.roundIDset.Do(func() {
		pass = false
	})
	if !pass {
		t.Error("Round ID could be set again, because the Once hadn't" +
			" been run yet")
	}

	// The round ID should be set
	if p.roundID != roundId {
		t.Error("Round ID wasn't set correctly")
	}

	// The state group should be the one we passed, and the phase state should
	// be Initialized
	if g != p.stateGroup {
		t.Error("State group wasn't set correctly")
	}
	if p.stateIndex != 0 {
		t.Error("State index wasn't set as expected")
	}
	if p.GetState() != Initialized {
		t.Error("State wasn't set to Initialized")
	}
}

// We can't use real graphs from realtime or precomputation phases, because
// they import Round and that causes an import cycle.
func initMockGraph(gg services.GraphGenerator) *services.Graph {
	return gg.NewGraph("MockGraph", nil)
}

func TestNew(t *testing.T) {
	timeout := 50 * time.Second
	// Testing whether the graph error handler is reachable is outside of the
	// scope of this test
	g := initMockGraph(services.NewGraphGenerator(1, nil, 1, 1, 1))
	pass := false
	phase := New(g, REAL_PERMUTE, func(phase *Phase,
		nal *services.NodeAddressList, getSlot GetChunk, getMessage GetMessage) {
		pass = true
	}, timeout)
	phase.GetTransmissionHandler()(nil, nil, nil, nil)
	if !pass {
		t.Error("Transmission handler was unreachable from Phase")
	}
	if phase.GetGraph() != g {
		t.Error("Graph wasn't set")
	}
	if phase.GetType() != REAL_PERMUTE {
		t.Error("Type wasn't set")
	}
	if phase.GetTimeout() != timeout {
		t.Error("Timeout wasn't set")
	}
}
