package io

import (
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
)

// Address of the subsequent server in the config file
// TODO remove this in favor of a better system
var NextServer string

// Boolean value for whether we are the last server
// TODO remove this in favor of a better system
var IsLastNode bool

// Struct implementing mixserver.ServerHandler interface
type ServerImpl struct {
	// Pointer to the global map of RoundID -> Rounds
	Rounds *globals.RoundMap
}

// Get the respective channel for the given roundId and chanId combination
func (s ServerImpl) GetChannel(roundId string, chanId globals.Phase) chan<- *services.Slot {
	return s.Rounds.GetRound(roundId).GetChannel(chanId)
}
