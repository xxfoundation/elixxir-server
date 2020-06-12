////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

// ReceivePostRoundPublicKey.go contains the handler for PostRoundPublicKey

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/phase"
	"time"
)

// ReceivePostRoundPublicKey from last node and sets it for the round
// for each node. Also starts precomputation decrypt phase with a
// batch
func ReceivePostRoundPublicKey(instance *internal.Instance,
	pk *mixmessages.RoundPublicKey, auth *connect.Auth) error {
	curActivity, err := instance.GetStateMachine().WaitFor(250*time.Millisecond, current.PRECOMPUTING)
	if err != nil {
		return errors.WithMessagef(err, errFailedToWait, current.PRECOMPUTING.String())
	}
	if curActivity != current.PRECOMPUTING {
		return errors.Errorf(errCouldNotWait, current.PRECOMPUTING.String())
	}

	roundID := id.Round(pk.Round.ID)
	rm := instance.GetRoundManager()
	r, err := rm.GetRound(roundID)
	if err != nil {
		return errors.WithMessagef(err, "Failed to retrieve round %+v", roundID)
	}

	// Verify that auth is good and sender is last node
	expectedID := r.GetTopology().GetLastNode()
	if !auth.IsAuthenticated || !auth.Sender.GetId().Cmp(expectedID) {
		jww.INFO.Printf("[%v]: RID %d ReceivePostRoundPublicKey failed auth "+
			"(expected ID: %s, received ID: %s, auth: %v)",
			instance, roundID, expectedID, auth.Sender.GetId(),
			auth.IsAuthenticated)
		return connect.AuthError(auth.Sender.GetId())
	}

	jww.INFO.Printf("[%v]: RID %d PostRoundPublicKey START", instance,
		roundID)

	tag := phase.PrecompShare.String() + "Verification"

	r, p, err := rm.HandleIncomingComm(roundID, tag)
	if err != nil {
		roundErr := errors.Errorf("[%v]: Error on reception of "+
			"PostRoundPublicKey comm, should be able to return: \n %+v",
			instance, err)
		return roundErr
	}
	p.Measure(measure.TagVerification)

	err = PostRoundPublicKey(instance.GetConsensus().GetCmixGroup(), r.GetBuffer(), pk)
	if err != nil {
		roundErr := errors.Errorf("[%v]: Error on posting PostRoundPublicKey "+
			"to io, should be able to return: %+v", instance, err)
		return roundErr
	}

	jww.INFO.Printf("[%v]: RID %d PostRoundPublicKey PK is: %s",
		instance, roundID, r.GetBuffer().CypherPublicKey.Text(16))

	p.UpdateFinalStates()

	jww.INFO.Printf("[%v]: RID %d PostRoundPublicKey END", instance,
		roundID)

	if r.GetTopology().IsFirstNode(instance.GetID()) {
		// We need to make a fake batch here because
		// we start the precomputation decrypt phase
		// afterwards.
		// This phase needs values of 1 for the keys & cypher
		// so we can apply modular multiplication afterwards.
		// Without this the ElGamal cryptop would need to
		// support this edge case.

		batchSize := r.GetBuffer().GetBatchSize()
		blankBatch := &mixmessages.Batch{}

		blankBatch.Round = pk.Round
		blankBatch.FromPhase = int32(phase.PrecompDecrypt)
		blankBatch.Slots = make([]*mixmessages.Slot, batchSize)

		for i := uint32(0); i < batchSize; i++ {
			blankBatch.Slots[i] = &mixmessages.Slot{
				EncryptedPayloadAKeys:     []byte{1},
				EncryptedPayloadBKeys:     []byte{1},
				PartialPayloadACypherText: []byte{1},
				PartialPayloadBCypherText: []byte{1},
			}
		}
		decrypt, err := r.GetPhase(phase.PrecompDecrypt)
		if err != nil {
			roundErr := errors.Errorf("Error on first node PostRoundPublicKey "+
				"comm, should be able to get decrypt phase: %+v", err)
			instance.ReportRoundFailure(roundErr, instance.GetID(), roundID)
		}

		jww.INFO.Printf("[%v]: RID %d PostRoundPublicKey FIRST NODE START PHASE \"%s\"", instance,
			roundID, decrypt.GetType())

		queued :=
			decrypt.AttemptToQueue(instance.GetResourceQueue().GetPhaseQueue())

		decrypt.Measure(measure.TagReceiveOnReception)

		if !queued {
			roundErr := errors.Errorf("Error on first node PostRoundPublicKey " +
				"comm, should be able to queue decrypt phase")
			instance.ReportRoundFailure(roundErr, instance.GetID(), roundID)
		}
		err = PostPhase(decrypt, blankBatch)

		if err != nil {
			roundErr := errors.Errorf("Error on first node PostRoundPublicKey "+
				"comm, should be able to post to decrypt phase: %+v", err)
			instance.ReportRoundFailure(roundErr, instance.GetID(), roundID)
		}
	}
	return nil
}
