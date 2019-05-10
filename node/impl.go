////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package io impl.go implements server utility functions needed to work
// with the comms library
package node

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/io"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/phase"
	"time"
)

// NewImplementation creates a new implementation of the server.
// When a function is added to comms, you'll need to point to it here.
func NewImplementation(instance *server.Instance) *node.Implementation {
	rm := instance.GetRoundManager()
	impl := node.NewImplementation()
	//impl.Functions.RoundtripPing = RoundtripPing
	//impl.Functions.GetServerMetrics = ServerMetrics
	//impl.Functions.CreateNewRound = NewRound
	//impl.Functions.StartRealtime = StartRealtime
	impl.Functions.GetRoundBufferInfo = func() (int, error) {
		return io.GetRoundBufferInfo(instance.GetCompletedPrecomps(),
			time.Second)
	}
	// FIXME: Should handle error and return Ack
	impl.Functions.PostPhase = func(batch *mixmessages.Batch) {

		_, p, err := rm.HandleIncomingComm(id.Round(batch.Round.ID), phase.Type(batch.ForPhase).String())
		if err != nil {
			jww.ERROR.Panicf("Error on comm, should be able to return: %+v", err)
		}
		err = io.PostPhase(p, batch)
		if err != nil {
			jww.ERROR.Panicf("Error on PostPhase comm, should be able to return: %+v", err)
		}
	}

	//impl.Functions.PostRoundPublicKey =
	//impl.Functions.RequestNonce = RequestNonce
	//impl.Functions.ConfirmRegistration = ConfirmRegistration
	//impl.Functions.PostPrecompResult = PostPrecompResult
	return impl
}
