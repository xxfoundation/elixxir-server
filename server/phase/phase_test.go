package phase

import (
	"fmt"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/services"
	"testing"
	"time"
)

// GETTER TESTS
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
	p := Phase{get: func() State {
        return Available
	},}
	if p.GetState() != state {
		t.Error("State from function was different than expected")
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

// Proves that Phase Cmp only returns true when the phases are the same
func TestPhase_Cmp(t *testing.T) {
	phaseType := uint32(2)
	roundID := id.Round(258)
	p := &Phase{
		roundID: roundID,
		tYpe:    Type(phaseType),
	}

	p2 := &Phase{
		roundID: roundID + 1,
		tYpe:    Type(phaseType + 1),
	}

	if !p.Cmp(p) {
		t.Error("Phase.Cmp: Phases are the same, returned that they are different")
	}

	if p.Cmp(p2) {
		t.Error("Phase.Cmp: Phases are different, returned that they are the same")
	}
}

func TestPhase_Stringer(t *testing.T) {
	phaseType := uint32(2)
	roundID := id.Round(258)
	p := &Phase{
		roundID: roundID,
		tYpe:    Type(phaseType),
	}

	p2 := &Phase{
		roundID: roundID + 1,
		tYpe:    Type(phaseType + 1),
	}

	pStr := fmt.Sprintf("phase.Phase{roundID: %v, phaseType: %s}",
		p.roundID, p.tYpe)

	p2Str := fmt.Sprintf("phase.Phase{roundID: %v, phaseType: %s}",
		p2.roundID, p2.tYpe)

	if p.String() != pStr {
		t.Errorf("Phase.String: Returned incorrect string, Expected: %s, Recieved: %s",
			pStr, p)
	}

	if p2.String() != p2Str {
		t.Errorf("Phase.String: Returned incorrect string, Expected: %s, Recieved: %s",
			p2Str, p2)
	}
}

func TestPhase_ReadyToReceiveData(t *testing.T) {
	state := Initialized
	p := Phase{get: func() State {
		return state
	},}
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
	var p Phase

	roundId := id.Round(55)
	state := Initialized
	p.ConnectToRound(roundId, func(to State) bool {
		state = to
        return true
	}, func() State {
		return state
	})

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

	// Getting the state should return Initialized
	if p.GetState() != Initialized {
		t.Error("State wasn't set to Initialized")
	}
	// We should be able to change the state with the function we passed
	p.IncrementStateToRunning()
	if p.GetState() != Running {
		t.Error("After changing the state, it wasn't set to Running")
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
