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
		//Check if the operation can be done and get the correct phase if it can
		_, p, err := rm.HandleIncomingComm(id.Round(batch.Round.ID), phase.Type(batch.ForPhase).String())
		if err != nil {
			jww.ERROR.Panicf("Error on comm, should be able to return: %+v", err)
		}

		//queue the phase to be operated on if it is not queued yet
		if p.AttemptTransitionToQueued() {
			instance.GetResourceQueue().UpsertPhase(p)
		}

		//send the data to the phase
		err = io.PostPhase(p, batch)
		if err != nil {
			jww.ERROR.Panicf("Error on PostPhase comm, should be able to return: %+v", err)
		}
	}

	// Receive round public key from last node and sets it for the round for each node.
	// Also starts precomputation decrypt phase with a batch
	impl.Functions.PostRoundPublicKey = func(pk *mixmessages.RoundPublicKey) {

		r, p, err := instance.HandleIncomingPhaseVerification(id.Round(pk.Round.ID), phase.PrecompShare)
		if err != nil {
			jww.ERROR.Panicf("Error on comm, should be able to return: %+v", err)
		}

		err = PostRoundPublicKey(instance.GetGroup(), r.GetBuffer(), pk)
		if err != nil {
			jww.ERROR.Panicf("Error on PostRoundPublicKey comm, should be able to return: %+v", err)
		}

		instance.GetResourceQueue().DenotePhaseCompletion(p)

		if r.GetNodeIDList().IsFirstNode() {
			// Make fake batch
			fakeBatch := &mixmessages.Batch{}
			impl.Functions.PostPhase(fakeBatch)
		}

	}

	impl.Functions.GetCompletedBatch = func() (batch *mixmessages.Batch, e error) {
		return io.GetCompletedBatch(instance.GetCompletedBatchQueue(), time.Second)
	}
	//impl.Functions.PostRoundPublicKey =
	impl.Functions.RequestNonce = func(salt, Y, P, Q, G, hash, R, S []byte) ([]byte, error) {
		return io.RequestNonce(instance, salt, Y, P, Q, G, hash, R, S)
	}
	impl.Functions.ConfirmRegistration = func(hash, R, S []byte) ([]byte, []byte, []byte,
		[]byte, []byte, []byte, []byte, error) {
		return io.ConfirmRegistration(instance, hash, R, S)
	}
	//impl.Functions.PostPrecompResult = PostPrecompResult
	return impl
}
