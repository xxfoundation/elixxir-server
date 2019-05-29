////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package node

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/io"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/phase"
)

func ReceivePostPhase(batch *mixmessages.Batch, instance *server.Instance) {

	rm := instance.GetRoundManager()

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

// Receive PostNewBatch comm from the gateway
// This should include an entire new batch that's ready for realtime processing
func ReceivePostNewBatch(instance *server.Instance,
	newBatch *mixmessages.Batch) error {
	// This shouldn't block,
	// and should return an error if there's no round available
	// You'd want to return this error in the Ack that's available for the
	// return value of the PostNewBatch comm.
	r, ok := instance.GetCompletedPrecomps().Pop()
	if !ok {
		return errors.New("ReceivePostNewBatch(): No precomputation available")
	}
	newBatch.Round.ID = uint64(r.GetID())
	newBatch.ForPhase = int32(phase.RealDecrypt)
	_, p, err := instance.GetRoundManager().HandleIncomingComm(r.GetID(),
		phase.RealDecrypt.String())
	if err != nil {
		jww.ERROR.Panicf("Error handling incoming PostNewBatch comm: %v", err)
	}

	// Queue the phase if it hasn't been done yet
	// Not sure if this is necessary
	// TODO Try commenting this out and see if it breaks the integration test
	//  you know, for fun
	if p.AttemptTransitionToQueued() {
		instance.GetResourceQueue().UpsertPhase(p)
	}

	for i := 0; i < len(newBatch.Slots); i++ {
		err := p.Input(uint32(i), newBatch.Slots[i])
		if err != nil {
			// All of the slots that didn't make it for some reason should
			// get put in a list so the gateway can tell the clients that there
			// was a problem
			// In the meantime, we're just logging the error
			jww.ERROR.Print(errors.Wrapf(err,
				"Slot %v failed for realtime decrypt.", i))
		}
	}
	// TODO send all the slot IDs that didn't make it back to the gateway
	return nil
}

// Receive round public key from last node and sets it for the round for each node.
// Also starts precomputation decrypt phase with a batch
func ReceivePostRoundPublicKey(instance *server.Instance,
	pk *mixmessages.RoundPublicKey, impl *node.Implementation) {

	rm := instance.GetRoundManager()

	tag := phase.PrecompShare.String() + "Verification"
	r, p, err := rm.HandleIncomingComm(id.Round(pk.Round.ID), tag)
	if err != nil {
		jww.ERROR.Panicf("Error on comm, should be able to return: %+v", err)
	}

	// Queue the phase to be operated on if it is not queued yet
	// Why does this need to be done? Wouldn't the phase have already been
	// run before the verification step happens?
	if p.AttemptTransitionToQueued() {
		instance.GetResourceQueue().UpsertPhase(p)
	}

	err = io.PostRoundPublicKey(instance.GetGroup(), r.GetBuffer(), pk)
	if err != nil {
		jww.ERROR.Panicf("Error on PostRoundPublicKey comm, should be able to return: %+v", err)
	}

	instance.GetResourceQueue().DenotePhaseCompletion(p)

	if r.GetTopology().IsFirstNode(instance.GetID()) {
		// We need to make a fake batch here because
		// we start the precomputation decrypt phase
		// afterwards.
		// This phase needs values of 1 for the keys & cypher
		// so we can apply modular multiplication afterwards.
		// Without this the ElGamal cryptop would need to
		// support this edge case.

		batchSize := r.GetBuffer().GetBatchSize()
		fakeBatch := &mixmessages.Batch{}

		fakeBatch.Round = pk.Round
		fakeBatch.ForPhase = int32(phase.PrecompDecrypt)
		fakeBatch.Slots = make([]*mixmessages.Slot, batchSize)

		for i := uint32(0); i < batchSize; i++ {
			fakeBatch.Slots[i] = &mixmessages.Slot{
				EncryptedMessageKeys:            []byte{1},
				EncryptedAssociatedDataKeys:     []byte{1},
				PartialMessageCypherText:        []byte{1},
				PartialAssociatedDataCypherText: []byte{1},
			}
		}

		impl.Functions.PostPhase(fakeBatch)

	}
}

func ReceivePostPrecompResult(instance *server.Instance, roundID uint64,
	slots []*mixmessages.Slot) error {
	rm := instance.GetRoundManager()

	tag := phase.PrecompReveal.String() + "Verification"
	r, p, err := rm.HandleIncomingComm(id.Round(roundID), tag)
	if err != nil {
		jww.ERROR.Panicf("Error on comm, should be able to return: %+v", err)
	}
	err = io.PostPrecompResult(r.GetBuffer(), instance.GetGroup(), slots)
	if err != nil {
		return errors.Wrapf(err,
			"Couldn't post precomp result for round %v", roundID)
	}
	instance.GetResourceQueue().DenotePhaseCompletion(p)
	// Now, this round has completed this precomputation,
	// so we can push it on the precomp queue if this is the first node
	if r.GetTopology().IsFirstNode(instance.GetID()) {
		instance.GetCompletedPrecomps().Push(r)
	}
	return nil
}

func ReceiveFinishRealtime(instance *server.Instance,
	message *mixmessages.RoundInfo) error {
	rm := instance.GetRoundManager()
	roundID := message.ID
	tag := phase.RealPermute.String() + "Verification"
	_, p, err := rm.HandleIncomingComm(id.Round(roundID), tag)
	if err != nil {
		jww.ERROR.Panicf("Error on comm, should be able to return: %+v", err)
	}
	err = io.FinishRealtime(rm, id.Round(roundID))
	if err != nil {
		return errors.Wrapf(err,
			"Couldn't finish realtime for round %v", roundID)
	}
	instance.GetResourceQueue().DenotePhaseCompletion(p)
	return nil
}
