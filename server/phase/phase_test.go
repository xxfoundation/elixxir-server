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
