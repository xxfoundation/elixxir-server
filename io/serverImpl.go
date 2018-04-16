////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/privategrity/server/cryptops/realtime"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
)

// Address of the subsequent server in the config file
// TODO remove this in favor of a better system
var NextServer string

// List of server addresses
// TODO remove this please thanks
var Servers []string

// These channels are used by LastNode to control when realtime and
// precomutation are kicked off
var RoundCh chan *string                  // Strings identifying rounds to be used
var MessageCh chan *realtime.RealtimeSlot // Message queuing

// Struct implementing mixserver.ServerHandler interface
type ServerImpl struct {
	// Pointer to the global map of RoundID -> Rounds
	Rounds *globals.RoundMap
}

// Get the respective channel for the given roundId and chanId combination
func (s ServerImpl) GetChannel(roundId string, chanId globals.Phase) chan<- *services.Slot {
	round := s.Rounds.GetRound(roundId)
	curPhase := round.GetPhase()
	if chanId != curPhase {
		jww.FATAL.Panicf("Round %s trying to start phase %s, but on phase %s!",
			roundId, chanId.String(), curPhase.String())
	}
	return round.GetChannel(chanId)
}

// Set the CypherPublicKey for the server to the given value
func (s ServerImpl) SetPublicKey(roundId string, newKey []byte) {
	s.Rounds.GetRound(roundId).CypherPublicKey.SetBytes(newKey)
}
