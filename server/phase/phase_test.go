////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package phase

import (
	"fmt"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/services"
	"strings"
	"testing"
	"time"
)

// GETTER TESTS
func TestPhase_GetGraph(t *testing.T) {
	g := services.Graph{}
	p := phase{
		graph: &g,
	}
	if p.GetGraph() != &g {
		t.Error("phase graphs were different")
	}
}

func TestPhase_GetRoundID(t *testing.T) {
	r := id.Round(562359865894179)
	p := phase{
		roundID: r,
	}
	if p.GetRoundID() != r {
		t.Error("Round ID was different")
	}
}

func TestPhase_GetTimeout(t *testing.T) {
	timeout := 580 * time.Second
	p := phase{
		timeout: timeout,
	}
	if p.GetTimeout() != timeout {
		t.Error("Timeout was different")
	}
}

func TestPhase_GetTransmissionHandler(t *testing.T) {
	pass := false
	handler := func(network *node.NodeComms, batchSize uint32,
		roundId id.Round, phaseTy Type, getSlot GetChunk,
		getMessage GetMessage, nodes *circuit.Circuit, nid *id.Node, measure Measure) error {
		pass = true
		return nil
	}
	p := phase{
		transmissionHandler: handler,
	}
	// This call should set pass to true
	err := p.GetTransmissionHandler()(nil, 0, 0, 0, nil, nil, nil, nil, nil)

	if err != nil {
		t.Errorf("Transmission handler returned an error, how!? %+v", err)
	}

	if !pass {
		t.Error("Didn't get the correct transmission handler")
	}
}

func TestPhase_GetState(t *testing.T) {
	state := Active
	p := phase{getState: func() State {
		return Active
	}}
	if p.GetState() != state {
		t.Error("State from function was different than expected")
	}
}

func TestPhase_GetType(t *testing.T) {
	phaseType := PrecompGeneration
	p := phase{tYpe: phaseType}
	if p.GetType() != phaseType {
		t.Error("Type was different")
	}
}

// Other tests prove that the various fields that should be set or compared
// are set or compared correctly

// Proves that phase Cmp only returns true when the phases are the same
func TestPhase_Cmp(t *testing.T) {
	phaseType := uint32(2)
	roundID := id.Round(258)
	p := &phase{
		roundID: roundID,
		tYpe:    Type(phaseType),
	}

	p2 := &phase{
		roundID: roundID + 1,
		tYpe:    Type(phaseType + 1),
	}

	if !p.Cmp(p) {
		t.Error("phase.Cmp: Phases are the same, returned that they are different")
	}

	if p.Cmp(p2) {
		t.Error("phase.Cmp: Phases are different, returned that they are the same")
	}
}

func TestPhase_Stringer(t *testing.T) {
	phaseType := uint32(2)
	roundID := id.Round(258)
	p := &phase{
		roundID: roundID,
		tYpe:    Type(phaseType),
	}

	p2 := &phase{
		roundID: roundID + 1,
		tYpe:    Type(phaseType + 1),
	}

	pStr := fmt.Sprintf("phase.phase{roundID: %v, phaseType: %s}",
		p.roundID, p.tYpe)

	p2Str := fmt.Sprintf("phase.phase{roundID: %v, phaseType: %s}",
		p2.roundID, p2.tYpe)

	if p.String() != pStr {
		t.Errorf("phase.String: Returned incorrect string, Expected: %s, Recieved: %s",
			pStr, p)
	}

	if p2.String() != p2Str {
		t.Errorf("phase.String: Returned incorrect string, Expected: %s, Recieved: %s",
			p2Str, p2)
	}
}

func TestPhase_ConnectToRound(t *testing.T) {

	timeout := 50 * time.Second

	pFace := New(Definition{nil, RealPermute, nil,
		timeout, false})

	p := pFace.(*phase)

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
		t.Errorf("phase connected should be initialized to 0")
	}

	if p.transitionToState != nil {

		t.Errorf("transitionToState should be initialized ot nil")
	}

	// Call connect to round on phase with round and set & get state handlers
	p.ConnectToRound(roundId, setState, getState)

	if *p.connected != 1 {
		t.Errorf("phase connected should be incremented from 0 to 1")
	}

	// The round ID should be set to correct value
	if p.roundID != roundId {
		t.Error("Round ID wasn't set correctly")
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
	g := initMockGraph(services.NewGraphGenerator(1, nil,
		1, 1, 1))
	pass := false

	transmit := func(network *node.NodeComms, batchSize uint32,
		roundId id.Round, phaseTy Type, getSlot GetChunk,
		getMessage GetMessage, nodes *circuit.Circuit, nid *id.Node, measure Measure) error {
		pass = true
		return nil
	}

	phase := New(Definition{g, RealPermute, transmit,
		timeout, false})
	err := phase.GetTransmissionHandler()(nil, 0, 0,
		0, nil, nil, nil, nil, nil)

	if err != nil {
		t.Errorf("Transmission handler returned an error, how!? %+v", err)
	}

	if !pass {
		t.Error("Transmission handler was unreachable from phase")
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

// Test that the function does not break calculating delta when a previous
// metric does not exist
func TestPhase_Measure(t *testing.T) {
	p := &phase{
		roundID: 0,
		tYpe: RealPermute,
	}

	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("test%d", i)
		r := getMeasureInfo(p, name)
		rs := strings.Split(r, "\t")

		if len(rs) != 6 {
			t.Errorf("Measure did not return enough variables in log for %s."+
				"\n\tExpected: %d vars\n\tGot:      %d vars",
				name, 5, len(rs)-1)
		}

		if rs[1] != "round ID: 0\n" {
			t.Errorf("Measure did not return correct round ID."+
				"\n\tExpected: %q\n\tGot:      %q",
				"round ID: 0\n", rs[1])
		}

		if rs[2] != "phase: 6\n" {
			t.Errorf("Measure did not return correct phase.\n\t" +
				"Expected: %q\n\tGot:      %q",
				"phase: 6\n", rs[2])
		}

		if rs[3] != fmt.Sprintf("tag: test%d\n", i) {
			t.Errorf("Measure did not return correct tag.\n\t" +
				"Expected: %q\n\tGot:      %q",
				fmt.Sprintf("tag: test%d\n", i), rs[3])
		}

		tstest := strings.SplitN(rs[4], " ", 2)
		if len(tstest) != 2 {
			t.Errorf("Measure returned a delta that parsed to more than two strings in space slit." +
				"\n\tGot: %q", rs[4])
		}
		// We have to define our own time format because Go doesn't have one for
		// the format outputted by timestamp.String()
		tf := "2006-01-02 15:04:05.99 -0700 MST"
		// Use some magic to remove the "m=+0.004601744" (example) part of the
		// outputted timestamp, since time.Parse() can't understand it
		ts := strings.Split(tstest[1], " ")
		ts = ts[:len(ts) - 1]
		// Join the newly cut string together with space separators and test it
		ts2 := strings.Join(ts, " ")
		_, err := time.Parse(tf, strings.TrimSpace(ts2))
		if err != nil {
			t.Errorf("Measure returned un-parsable timestamp\n\tGot: %q", rs[4])
		}

		deltatest := strings.Split(rs[5], " ")
		if len(deltatest) != 2 {
			t.Errorf("Measure returned a delta that parsed to more than two strings in space slit." +
				"\n\tGot: %q", rs[5])
		}
		if i == 0 && deltatest[1] != "0s" {
			t.Errorf("Measure returned a delta that isn't 0s for first measurement." +
				"\n\tExpected: %q\n\tGot:      %q",
				"0s", deltatest[1])
		}
		delta, err := time.ParseDuration(deltatest[1])
		if err != nil {
			t.Errorf("Measure returned un-parsable delta.\n\tGot: \"%s\"", deltatest[1])
			return
		}
		if delta.Nanoseconds() < 0 {
			t.Errorf("Measure returned a negative duration.\n\tGot: \"%s\"", deltatest[1])
		}
	}
}
