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
	"gitlab.com/elixxir/server/server/round"
)

//ReceiveCreateNewRound receives the create new round signal and creates the round
func ReceiveCreateNewRound(instance *server.Instance, message *mixmessages.RoundInfo) error {
	roundID := id.Round(message.ID)

	//Build the components of the round
	phases, phaseResponses := NewRoundComponents(instance.GetGraphGenerator(),
		instance.GetTopology(), instance.GetID(), &instance.LastNode, instance.GetBatchSize())
	//Build the round
	rnd := round.New(instance.GetGroup(), instance.GetUserRegistry(), roundID,
		phases, phaseResponses, instance.GetTopology(), instance.GetID(),
		instance.GetBatchSize())
	//Initialize crypto fields for round
	rnd.GetBuffer().InitCryptoFields(instance.GetGroup())
	//Add the round to the manager
	instance.GetRoundManager().AddRound(rnd)

	return nil
}

// ReceivePostPhase handles the state checks and edge checks of receiving a
// phase operation
func ReceivePostPhase(batch *mixmessages.Batch, instance *server.Instance) {

	rm := instance.GetRoundManager()

	//Check if the operation can be done and get the correct phase if it can
	_, p, err := rm.HandleIncomingComm(id.Round(batch.Round.ID), phase.Type(batch.FromPhase).String())
	if err != nil {
		jww.ERROR.Panicf("Error on comm, should be able to return: %+v", err)
	}

	//queue the phase to be operated on if it is not queued yet
	if p.AttemptTransitionToQueued() {
		instance.GetResourceQueue().UpsertPhase(p)
	}

	//HACK HACK HACK
	//The share phase needs a batchsize of 1, when it recieves from generation
	//on the first node this will do the conversion on the batch
	if p.GetType() == phase.PrecompShare && len(batch.Slots) != 1 {
		batch.Slots = batch.Slots[:1]
	}

	//send the data to the phase
	err = io.PostPhase(p, batch)

	if err != nil {
		jww.ERROR.Panicf("Error on PostPhase comm, should be able to return: %+v", err)
	}

}

// ReceiveStreamPostPhase handles the state checks and edge checks of receiving a
// phase operation
func ReceiveStreamPostPhase(streamServer mixmessages.Node_StreamPostPhaseServer, instance *server.Instance) error {

	rm := instance.GetRoundManager()

	batchInfo, err := node.GetPostPhaseStreamHeader(streamServer)
	if err != nil {
		return err
	}

	// Check if the operation can be done and get the correct phase if it can
	_, p, err := rm.HandleIncomingComm(id.Round(batchInfo.Round.ID), phase.Type(batchInfo.FromPhase).String())
	if err != nil {
		jww.ERROR.Panicf("Error on comm, should be able to return: %+v", err)
	}

	//queue the phase to be operated on if it is not queued yet
	if p.AttemptTransitionToQueued() {
		instance.GetResourceQueue().UpsertPhase(p)
	}

	//HACK HACK HACK
	//The share phase needs a batchsize of 1, when it recieves from generation
	//on the first node this will do the conversion on the batch
	if p.GetType() == phase.PrecompShare && batchInfo.BatchSize != 1 {
		batchInfo.BatchSize = 1
	}

	return io.StreamPostPhase(p, batchInfo.BatchSize, streamServer)

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
		err := errors.New("ReceivePostNewBatch(): No precomputation available")
		// This round should be at a state where its precomp is complete.
		// So, we might want more than one phase,
		// since it's at a boundary between phases.
		jww.ERROR.Print(err)
		return err
	}
	newBatch.Round.ID = uint64(r.GetID())
	newBatch.FromPhase = int32(phase.RealDecrypt)
	_, p, err := instance.GetRoundManager().HandleIncomingComm(r.GetID(),
		phase.RealDecrypt.String())
	if err != nil {
		jww.ERROR.Panicf("Error handling incoming PostNewBatch comm: %v", err)
	}

	// Queue the phase if it hasn't been done yet
	if p.AttemptTransitionToQueued() {
		instance.GetResourceQueue().UpsertPhase(p)
	}

	for i := 0; i < len(newBatch.Slots); i++ {
		err := p.Input(uint32(i), newBatch.Slots[i])
		if err != nil {
			// TODO All of the slots that didn't make it for some reason should
			//  get put in a list so the gateway can tell the clients that there
			//  was a problem
			//  In the meantime, we're just logging the error
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
		fakeBatch.FromPhase = int32(phase.PrecompDecrypt)
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

// ReceivePostPrecompResult handles the state checks and edge checks of
// receiving the result of the precomputation
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

// ReceiveFinishRealtime handles the state checks and edge checks of
// receiving the signal that the realtime has completed
func ReceiveFinishRealtime(instance *server.Instance,
	msg *mixmessages.RoundInfo) error {

	//check that the round should have finished and return it
	roundID := id.Round(msg.ID)

	rm := instance.GetRoundManager()

	tag := "Completed"
	rnd, _, err := rm.HandleIncomingComm(id.Round(roundID), tag)
	if err != nil {
		return err
	}

	//release the round's data
	rnd.GetBuffer().Erase()

	//delete the round from the manager
	rm.DeleteRound(roundID)

	//Send the finished signal on first node
	if rnd.GetTopology().IsFirstNode(instance.GetID()) {
		instance.FinishRound(roundID)
	}

	return nil
}
