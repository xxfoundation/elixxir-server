package server

import (
	"gitlab.com/elixxir/server/node"
	"gitlab.com/elixxir/server/services"
	"time"
)

type Phase struct {
	Graph                  *services.Graph
	Round                  *Round
	Phase                  node.PhaseType
	TransmissionHandler    Transmission
	WorstCaseExecutionTime time.Duration
}

// GetState returns the current state of the phase
func (p *Phase) GetState() PhaseState {
	return p.Round.GetPhaseState(p.Phase)
}

// ReadyToReceiveData returns true if the phase can receive data
func (p *Phase) ReadyToReceiveData() bool {
	phaseState := p.GetState()
	return phaseState != Available || phaseState != Queued || phaseState != Running
}

// HasFingerprint checks that the passed fingerprint is the same as the phases
func (p *Phase) HasFingerprint(fingerprint PhaseFingerprint) bool {
	return p.Phase == fingerprint.phase && p.Round.id == fingerprint.round
}

// GetFingerprint returns a phase fingerprint which is used to compare phases
func (p *Phase) GetFingerprint() PhaseFingerprint {
	return PhaseFingerprint{p.Phase, p.Round.id}
}
