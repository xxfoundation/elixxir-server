package round

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/io"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/phase"
)

func PostPhaseImpl(instance *server.Instance) func(batch *mixmessages.Batch) {

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

func PostRoundPublicKeyImpl(instance *server.Instance) func(pk *mixmessages.RoundPublicKey)  {
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

		batchSize := r.GetBuffer().GetBatchSize()
		if r.GetTopology().IsFirstNode(instance.GetID()) {
			// We need to make a fake batch here because
			// we start post phase afterwards and it
			// needs the identity values of 1 for the data
			// so we can apply modular multiplication.
			// Without this the El Gamal cryptop would need to
			// support this edge case.
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

			PostPhaseImpl(instance)()
		}
	}
}