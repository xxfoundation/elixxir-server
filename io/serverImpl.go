////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/server/cryptops/realtime"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/services"
)

// Address of the subsequent server in the config file
// TODO remove this in favor of a better system
var NextServer string

// List of server addresses
// TODO remove this please thanks
var Servers []string
var TimeUp int64

// These channels are used by LastNode to control when realtime and
// precomutation are kicked off
var RoundCh chan *string          // Strings identifying rounds to be used
var MessageCh chan *realtime.Slot // Message queuing

// Struct implementing node.ServerHandler interface
type ServerImpl struct {

	// Pointer to the global map of RoundID -> Rounds
	Rounds *globals.RoundMap
}

// Get the respective channel for the given roundID and chanId combination
func (s ServerImpl) GetChannel(roundID string, chanId globals.Phase) chan<- *services.Slot {
	round := s.Rounds.GetRound(roundID)
	curPhase := round.GetPhase()
	if chanId != curPhase && curPhase != globals.ERROR {
		jww.FATAL.Panicf("Round %s trying to start phase %s, but on phase %s!",
			roundID, chanId.String(), curPhase.String())
	}
	return round.GetChannel(chanId)
}

// Set the CypherPublicKey for the server to the given value
func (s ServerImpl) SetPublicKey(roundID string, newKey []byte) {
	s.Rounds.GetRound(roundID).CypherPublicKey.SetBytes(newKey)
}
