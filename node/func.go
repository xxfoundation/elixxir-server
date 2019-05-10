////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package node

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/io"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/phase"
)

func RoundtripPingFunc(instance *server.Instance) func(*mixmessages.TimePing) {
	return nil
}

func ServerMetricsFunc(instance *server.Instance) func(*mixmessages.ServerMetrics) {
	return nil
}

func NewRoundFunc(instance *server.Instance) func(message *mixmessages.RoundInfo) {
	return nil
}

func GetRoundBufferInfoFunc(instance *server.Instance) func() (int, error) {
	return nil
}

func PostPhaseFunc(instance *server.Instance) func(message *mixmessages.Batch) {

	rm := instance.GetRoundManager()

	return func(batch *mixmessages.Batch) {

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
}

// Receive round public key from last node and sets it for the round for each node.
// Also starts precomputation decrypt phase with a batch
func PostRoundPublicKeyFunc(instance *server.Instance,
	postPhaseFunc func(*server.Instance) func(message *mixmessages.Batch)) func(pk *mixmessages.RoundPublicKey) {

	rm := instance.GetRoundManager()

	return func(pk *mixmessages.RoundPublicKey) {

		tag := phase.Type(phase.PrecompShare).String() + "Verification"
		r, p, err := rm.HandleIncomingComm(id.Round(pk.Round.ID), tag)
		if err != nil {
			jww.ERROR.Panicf("Error on comm, should be able to return: %+v", err)
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

			postPhaseFunc(instance)(fakeBatch)
		}
	}
}

func PostPrecompResultFunc(instance *server.Instance) func(roundID uint64, slots []*mixmessages.Slot) error {
	return nil
}

func ConfirmRegistrationFunc(instance *server.Instance) func(hash []byte, R []byte, S []byte) ([]byte, []byte, []byte, []byte, []byte, []byte, []byte, error) {
	return nil
}

func RequestNonceFunc(instance *server.Instance) func(salt []byte, Y []byte, P []byte, Q []byte, G []byte, hash []byte, R []byte, S []byte) ([]byte, error) {
	return nil
}
