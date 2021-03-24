///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Package round manager.go provides a manager that keeps track of the
// round objects
package round

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/xx_network/primitives/id"
	"sync"
)

// Manager contains a pointer to the roundMap, which maps round id's to rounds
type Manager struct {
	roundMap    *sync.Map
	latestRound id.Round
}

// NewManager creates a new manager object with an empty round map
func NewManager() *Manager {
	rmap := sync.Map{}
	return &Manager{
		roundMap:    &rmap,
		latestRound: 0,
	}
}

// AddRound adds the round to the round manager's tracking
func (rm *Manager) AddRound(round *Round) {
	rm.latestRound = round.id
	rm.roundMap.Store(round.id, round)
}

func (rm *Manager) GetLatestRound() id.Round {
	return rm.latestRound
}

// GetRound returns the round if it exists, or an error if it doesn't
func (rm *Manager) GetRound(id id.Round) (*Round, error) {
	r, ok := rm.roundMap.Load(id)

	if !ok {
		return nil, errors.Errorf("Could not find Round ID: %d", id)
	}

	return r.(*Round), nil
}

// GetPhase checks that the phase type is correct and returns the correct
// phase object for the given Round ID. This does error checking
// as it is intended to be called from network handlers
func (rm *Manager) GetPhase(id id.Round, phaseTy int32) (phase.Phase, error) {
	// First, check that the phase type id # is valid
	if phaseTy < 0 || phaseTy >= int32(phase.NUM_PHASES) {
		return nil, errors.Errorf("Invalid phase Type Number: %d",
			phaseTy)
	}

	r, rErr := rm.GetRound(id)
	if rErr != nil {
		return nil, rErr
	}

	p, pErr := r.GetPhase(phase.Type(uint32(phaseTy)))
	if pErr != nil {
		return nil, pErr
	}

	return p, nil
}

// DeleteRound removes the round for this ID from the manager, if the
// manager is keeping track of it
func (rm *Manager) DeleteRound(id id.Round) {
	rm.roundMap.Delete(id)
}

// HandleIncomingComm looks up if a comm is valid and if it is, returns
// the associated round, phase (according to the round's response table)
// otherwise returns an error
func (rm *Manager) HandleIncomingComm(roundID id.Round, tag string) (*Round, phase.Phase, error) {
	// Get the round (with error checking) from the round manager
	r, err := rm.GetRound(roundID)
	if err != nil {
		return nil, nil, err
	}
	//Get the correct phase from the round based upon the response table
	p, err := r.HandleIncomingComm(tag)
	if err != nil {
		return nil, nil, err
	}

	return r, p, nil
}
