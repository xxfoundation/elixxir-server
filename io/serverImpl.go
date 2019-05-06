////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package io serverImpl.go implements server utility functions needed to work
// with the comms library
package io

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/services"
)

// NewServerImplementation creates a new implementation of the server.
// When a function is added to comms, you'll need to point to it here.
func NewServerImplementation(instance *server.Instance) *node.Implementation {
	impl := node.NewImplementation()
	//impl.Functions.RoundtripPing = RoundtripPing
	//impl.Functions.GetServerMetrics = ServerMetrics
	//impl.Functions.CreateNewRound = NewRound
	//impl.Functions.StartRealtime = StartRealtime
	//impl.Functions.GetRoundBufferInfo = GetRoundBufferInfo
	// FIXME: Should handle error and return Ack
	impl.Functions.PostPhase = func(batch *mixmessages.Batch) {
		phase, err := instance.HandleIncomingPhase(id.Round(batch.Round.ID), phase.Type(batch.ForPhase))
		if err != nil {
			jww.ERROR.Panicf("Error on comm, should be able to return: %+v", err)
		}
		err = PostPhase(phase, batch)
		if err != nil {
			jww.ERROR.Panicf("Error on comm, should be able to return: %+v", err)
		}
	}

	// Receive round public key from last node and sets it for the round for each node.
	// Also starts precomputation decrypt phase with a batch
	impl.Functions.PostRoundPublicKey = func(pk *mixmessages.RoundPublicKey) {

		phase, err := instance.HandleIncomingPhase(id.Round(pk.Round.ID), phase.PrecompShare)
		if err != nil {
			jww.ERROR.Panicf("Error on comm, should be able to return: %+v", err)
		}

		// Where do we receive this node address list from?
		// Do we need to loop over all nodes?
		var nal *services.NodeAddressList
		nodeAddrList := nal.GetAllNodesAddress()

		for _, nodeAddr := range nodeAddrList {


			// Handler sets the round public key in the round buffer
			err = PostRoundPublicKey(phase, fakeBatch, pk)

		}

		// Start precomputation decrypt phase with fakeBatch
		//	if isFirstNode{
		//		//built a fake fakeBatch input
		//		impl.Functions.PostPhase(fakeBatch)
		//
		//	}
	}

	//impl.Functions.RequestNonce = RequestNonce
	//impl.Functions.ConfirmRegistration = ConfirmRegistration
	//impl.Functions.PostPrecompResult = PostPrecompResult
	return impl
}
