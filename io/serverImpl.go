////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/node"
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
// precomputation are kicked off
var RoundCh chan *string          // Strings identifying rounds to be used
var MessageCh chan *realtime.Slot // Message queuing

// NewServerImplementation creates a new implementation of the server.
// When a function is added to comms, you'll need to point to it here.
func NewServerImplementation() *node.Implementation {
	impl := node.NewImplementation()
	impl.Functions.RoundtripPing = RoundtripPing
	impl.Functions.ServerMetrics = ServerMetrics
	impl.Functions.NewRound = NewRound
	impl.Functions.StartRound = StartRound
	impl.Functions.GetRoundBufferInfo = GetRoundBufferInfo
	impl.Functions.SetPublicKey = SetPublicKey
	impl.Functions.PrecompDecrypt = PrecompDecrypt
	impl.Functions.PrecompEncrypt = PrecompEncrypt
	impl.Functions.PrecompReveal = PrecompReveal
	impl.Functions.PrecompPermute = PrecompPermute
	impl.Functions.PrecompShare = PrecompShare
	impl.Functions.PrecompShareInit = PrecompShareInit
	impl.Functions.PrecompShareCompare = PrecompShareCompare
	impl.Functions.PrecompShareConfirm = PrecompShareConfirm
	impl.Functions.RealtimeDecrypt = RealtimeDecrypt
	impl.Functions.RealtimeEncrypt = RealtimeEncrypt
	impl.Functions.RealtimePermute = RealtimePermute
	impl.Functions.RequestNonce = RequestNonce
	impl.Functions.ConfirmNonce = ConfirmNonce
	return impl
}

// Get the respective channel for the given roundID and chanId combination
func GetChannel(roundID string, chanId globals.Phase) chan<- *services.Slot {
	round := globals.GlobalRoundMap.GetRound(roundID)
	curPhase := round.GetPhase()
	if chanId != curPhase && curPhase != globals.ERROR {
		jww.FATAL.Panicf("Round %s trying to start phase %s, but on phase %s!",
			roundID, chanId.String(), curPhase.String())
	}
	return round.GetChannel(chanId)
}

// Set the CypherPublicKey for the server to the given value
func SetPublicKey(roundID string, newKey []byte) {
	globals.GetGroup().SetBytes(globals.GlobalRoundMap.GetRound(roundID).CypherPublicKey, newKey)
}
