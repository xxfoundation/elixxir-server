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
	handler := func(batchSize uint32, roundId id.Round, phaseTy Type, getSlot GetChunk,
		getMessage GetMessage, nal *services.NodeAddressList) error {
		pass = true
		return nil
	}
	p := Phase{
		transmissionHandler: handler,
	}
	// This call should set pass to true
	err := p.GetTransmissionHandler()(0, 0, 0, nil, nil, nil)

	if err != nil {
		t.Errorf("Transmission handler returned an error, how!? %+v", err)
	}

	if !pass {
		t.Error("Didn't get the correct transmission handler")
	}
}

func TestPhase_GetState(t *testing.T) {
	state := Available
	p := Phase{getState: func() State {
		return Available
	}}
	if p.GetState() != state {
		t.Error("State from function was different than expected")
	}
}

func TestPhase_GetType(t *testing.T) {
	phaseType := PrecompGeneration
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

	p := Phase{
		getState: func() State {
			return state
		},
		connected: new(uint32),
	}

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
	state = Computed
	if p.ReadyToReceiveData() {
		t.Error("Running phase should not be ready to receive")
	}
	p.EnableVerification()
	if p.ReadyToReceiveData() {
		t.Error("Running phase should still not be ready to receive")
	}
	if !p.ReadyToVerify() {
		t.Error("Running phase should be ready to verify")
	}
	state = Verified
	if p.ReadyToVerify() {
		t.Error("Running phase should no longer be ready to verify")
	}
	if p.ReadyToReceiveData() {
		t.Error("Verified phase should not be ready to receive")
	}

}

func TestPhase_ConnectToRound(t *testing.T) {

	timeout := 50 * time.Second

	p := New(nil, RealPermute, nil, timeout)

	// Initial inputs to ConnectToRound shouldn't change after calls
	roundId := id.Round(55)
	state := Initialized
	setState := func(from, to State) bool {
		state = to
		return true
	}
	getState := func() State {
		return state
	}

	if *p.connected != 0 {
		t.Errorf("Phase connected should be initialized to 0")
	}

	if p.transitionToState != nil {

		t.Errorf("transitionToState should be initialized ot nil")
	}

	// Call connect to round on phase with round and set & get state handlers
	p.ConnectToRound(roundId, setState, getState)

	if *p.connected != 1 {
		t.Errorf("Phase connected should be incremented from 0 to 1")
	}

	// The round ID should be set to correct value
	if p.roundID != roundId {
		t.Error("Round ID wasn't set correctly")
	}

	if p.GetState() != Initialized {
		t.Error("State wasn't set to Initialized")
	}

	roundId2 := id.Round(85)
	state2 := Running
	setState2 := func(from, to State) bool {
		state2 = to
		return true
	}
	getState2 := func() State {
		return state2
	}
	// Call connect to round again on phase with round and set & get state handlers
	p.ConnectToRound(roundId2, setState2, getState2)

	if *p.connected != 2 {
		t.Errorf("Phase connected should be incremented from 1 to 2")
	}

	// The round ID should be set to correct value
	if p.roundID != roundId {
		t.Error("Round ID changed to incorrect value")
	}

	if p.GetState() != Initialized {
		t.Error("State was changed from Initialized to incorrect value ", p.GetState())
	}

	p.TransitionToRunning()

	// We should be able to change the state with the function we passed
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

	transmit := func(batchSize uint32, roundId id.Round, phaseTy Type, getSlot GetChunk,
		getMessage GetMessage, nal *services.NodeAddressList) error {
		pass = true
		return nil
	}

	phase := New(g, RealPermute, transmit, timeout)
	err := phase.GetTransmissionHandler()(0, 0, 0, nil, nil, nil)

	if err != nil {
		t.Errorf("Transmission handler returned an error, how!? %+v", err)
	}

	if !pass {
		t.Error("Transmission handler was unreachable from Phase")
	}
	if phase.GetGraph() != g {
		t.Error("Graph wasn't set")
	}
	if phase.GetType() != RealPermute {
		t.Error("Type wasn't set")
	}
	if phase.GetTimeout() != timeout {
		t.Error("Timeout wasn't set")
	}
}
