///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package io

// Contains handlers for StartSharePhase and SharePhasePiece

import (
	"bytes"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/signature"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// StartSharePhase is a reception handler for TransmitStartSharePhase.
// It does basic checks of the validity of the message and the sender.
// After checks are complete, it generates and transmits its own share by
// calling TransmitPhaseShare
func StartSharePhase(ri *pb.RoundInfo, auth *connect.Auth,
	instance *internal.Instance) error {
	// todo: check phase here, figure out how to do that. maybe use handlecomm func

	curActivity, err := instance.GetStateMachine().WaitFor(250*time.Millisecond, current.PRECOMPUTING)
	if err != nil {
		return errors.WithMessagef(err, errFailedToWait, current.PRECOMPUTING.String())
	}
	if curActivity != current.PRECOMPUTING {
		return errors.Errorf(errCouldNotWait, current.PRECOMPUTING.String())
	}

	// Get round from round manager
	roundID := id.Round(ri.ID)
	rm := instance.GetRoundManager()
	r, err := rm.GetRound(roundID)
	if err != nil {
		return errors.WithMessage(err, "Failed to get round")
	}

	// Check for proper authentication
	// and if the sender is the leader of the round
	topology := r.GetTopology()
	if !auth.IsAuthenticated || !topology.IsFirstNode(auth.Sender.GetId()) {
		jww.WARN.Printf("Error on PostPhase: "+
			"Attempted communication by %+v has not been authenticated: %s", auth.Sender, auth.Reason)
		return errors.WithMessage(connect.AuthError(auth.Sender.GetId()), auth.Reason)
	}

	//internal edge checking, independant of system
	// check and iterate within handleincoming comm

	// todo: check phase here, figure out how to do that. maybe use handlecomm func
	tag := phase.PrecompShare.String() // todo: change tag(?)
	_, p, err := rm.HandleIncomingComm(roundID, tag)
	if err != nil {
		return errors.Errorf("[%v]: Error on reception of "+
			"StartSharePhase comm, should be able to return: \n %+v",
			instance, err)
	}
	p.Measure(measure.TagReceiveOnReception)

	// Generate and sign the  message to be shared with the team
	err = TransmitPhaseShare(instance, r, nil)
	if err != nil {
		jww.FATAL.Panicf("Error on StartSharePhase: "+
			"Could not send our shared piece of the key: %v", err)
	}

	return nil
}

// SharePhasePiece is a reception handler for receiving a key generation share.
// It does basic validity checks on the share and the sender. If our node is not
// in the list of participants of the message, we generate a share off of the
// received piece and send that share all other teammates.
// We also update our state for this received message
func SharePhasePiece(piece *pb.SharePiece, auth *connect.Auth, instance *internal.Instance) error {

	if piece == nil || piece.Participants == nil || piece.Piece == nil {
		return errors.Errorf("Should not receive nil pieces in key generation")
	}

	// Check state machine for proper state
	curActivity, err := instance.GetStateMachine().WaitFor(250*time.Millisecond, current.PRECOMPUTING)
	if err != nil {
		return errors.WithMessagef(err, errFailedToWait, current.PRECOMPUTING.String())
	}
	if curActivity != current.PRECOMPUTING {
		return errors.Errorf(errCouldNotWait, current.PRECOMPUTING.String())
	}

	// Get round from round manager
	roundID := id.Round(piece.RoundID)
	rm := instance.GetRoundManager()
	r, err := rm.GetRound(roundID)
	if err != nil {
		return errors.WithMessage(err, "Failed to get round")
	}

	// todo: check phase here, figure out how to do that. maybe use handlecomm func
	tag := phase.PrecompShare.String() // todo: change tag
	_, p, err := rm.HandleIncomingComm(roundID, tag)
	if err != nil {
		return errors.Errorf("[%v]: Error on reception of "+
			"SharePhasePiece comm, should be able to return: \n %+v",
			instance, err)
	}
	p.Measure(tag)

	// Check for proper authentication and if the sender is in the round
	senderId := auth.Sender.GetId()
	if !auth.IsAuthenticated ||
		r.GetTopology().GetNodeLocation(senderId) == -1 {
		jww.WARN.Printf("Error on SharePhasePiece: "+
			"Attempted communication by %+v has not been authenticated: %s",
			auth.Sender, auth.Reason)
		return errors.WithMessage(connect.AuthError(auth.Sender.GetId()), auth.Reason)
	}

	// Handle the state of our round and update for received piece
	err = updateSharePieces(instance, r, senderId, piece)
	if err != nil {
		jww.FATAL.Panicf("Error on round [%d] SharePhasePiece: Could not update "+
			"new round shares: %s", r.GetID(), err)
	}

	// If we are aren't already a participant, we must send our share on this piece
	participants := piece.Participants
	if !isAlreadyParticipant(participants, instance.GetID().Bytes()) {
		// Develop and send our share
		err = TransmitPhaseShare(instance, r, piece)
		if err != nil {
			jww.FATAL.Panicf("Error on SharePhasePiece: "+
				"Could not send our shared piece of the key: %s", err)
		}

	}

	return nil
}

// involve incremnt shares. Going to have to get pulled out, on edge check
//

//

// updateSharePieces is a helper function which updates the state for a new
// round's phaseShare being received. If we have received a share from all
// we add to a list of final keys. If our final keys list is the size of the
// team, then we are done with sharePhase and check all messages and keys.
// If we don't enough final keys, we start a new sub-round of generating a key
func updateSharePieces(instance *internal.Instance, r *round.Round,
	originID *id.ID, piece *pb.SharePiece) error {
	// Add message to state
	r.AddPieceMessage(piece, originID)
	newAmountOfShares := r.IncrementShares()

	// Pull the participants and teamsize information
	participants := piece.Participants
	teamSize := r.GetTopology().Len()

	// Check if this shared piece has gone through all nodes
	if len(participants) == teamSize {
		// Add the key with all shares to our state
		grp := instance.GetConsensus().GetCmixGroup()
		roundKey := grp.NewIntFromBytes(piece.Piece)
		finalKeys := r.UpdateFinalKeys(roundKey)

		// Check if the amount of keys added to the list is equal to the teamsize.
		// ie shares originated from every node in the team has made the rounds
		// across the team to be generated as a final key
		if teamSize == len(finalKeys) {
			// Check that the every node has sent a share [teamsize^2] times
			if (teamSize * teamSize) != int(newAmountOfShares) {
				// When the amount of keys generated are (teamsize), the amount
				// of shares should also be (teamsize^2). If this is not the case,
				// something has gone wrong
				return errors.Errorf("Amount of shares [%d] does not reflect "+
					"every node sending a message expected amount of times. "+
					"Expected amount of shares [%d]",
					newAmountOfShares, teamSize*teamSize)

			}

			// If so, perform finalize the key generation phase
			err := finalizeKeyGeneration(instance, r, finalKeys)
			if err != nil {
				return errors.Errorf("Could not "+
					"finalize key generation: %s", err)
			}

		}
	}

	return nil
}

// finalizeKeyGeneration makes the final checks for the key generation.
// It checks that all received messages' have valid signatures from the sender
// and that all keys generated in every round of phaseShare matches
// Finally it sets the round buffer public key if that
func finalizeKeyGeneration(instance *internal.Instance, r *round.Round,
	finalKeys []*cyclic.Int) error {

	// Check signatures sent by every host
	if err := checkSignatures(r); err != nil {
		return err
	}

	// Check if all the keys generated in every sub-round match
	if err := checkFinalKeys(finalKeys); err != nil {
		return err
	}

	// Pull a key from the list (all should be identical from
	// the above check)
	finalKey := finalKeys[0].Bytes()

	// Double check that key is inside of group
	grp := instance.GetConsensus().GetCmixGroup()
	inside := grp.BytesInside(finalKey)
	if !inside {
		return services.ErrOutsideOfGroup
	}

	// Set round key now that everything is confirmed
	grp.SetBytes(r.GetBuffer().CypherPublicKey, finalKey)

	// Once done, transition from precompShare to precompDecrypt
	// Fixme: figure out how to properly transfer phases
	//if err := transitionToPrecompDecrypt(instance, r); err != nil {
	//	return errors.Errorf("Could not transition to precompDecrypt: %s", err)
	//}

	return nil
}

// transitionToPrecompDecrypt is a helper function which handles the business
// logic of transitioning from precompShare to precompDecrypt
func transitionToPrecompDecrypt(instance *internal.Instance, r *round.Round) error {

	rm := instance.GetRoundManager()
	topology := r.GetTopology()
	roundID := r.GetID()

	tag := phase.PrecompShare.String() + "Verification"
	r, p, err := rm.HandleIncomingComm(roundID, tag)
	if err != nil {
		roundErr := errors.Errorf("Error on reception of "+
			"SharePhasePiece comm, should be able to return: \n %+v", err)
		return roundErr
	}
	p.Measure(measure.TagVerification)

	jww.INFO.Printf("[%v]: RID %d SharePhasePiece PK is: %s",
		instance, roundID, r.GetBuffer().CypherPublicKey.Text(16))

	p.UpdateFinalStates()

	jww.INFO.Printf("[%v]: RID %d SharePhasePiece END", instance,
		roundID)

	if topology.IsFirstNode(instance.GetID()) {
		// We need to make a fake batch here because
		// we start the precomputation decrypt phase
		// afterwards.
		// This phase needs values of 1 for the keys & cypher
		// so we can apply modular multiplication afterwards.
		// Without this the ElGamal cryptop would need to
		// support this edge case.

		batchSize := r.GetBuffer().GetBatchSize()
		blankBatch := &pb.Batch{}

		blankBatch.Round = &pb.RoundInfo{
			ID: uint64(r.GetID()),
		}
		blankBatch.FromPhase = int32(phase.PrecompDecrypt)
		blankBatch.Slots = make([]*pb.Slot, batchSize)

		for i := uint32(0); i < batchSize; i++ {
			blankBatch.Slots[i] = &pb.Slot{
				EncryptedPayloadAKeys:     []byte{1},
				EncryptedPayloadBKeys:     []byte{1},
				PartialPayloadACypherText: []byte{1},
				PartialPayloadBCypherText: []byte{1},
			}
		}
		decrypt, err := r.GetPhase(phase.PrecompDecrypt)
		if err != nil {
			return errors.Errorf("Error on first node SharePhasePiece "+
				"comm, should be able to get decrypt phase: %+v", err)
		}

		jww.INFO.Printf("[%v]: RID %d SharePhasePiece FIRST NODE START PHASE \"%s\"", instance,
			roundID, decrypt.GetType())

		queued :=
			decrypt.AttemptToQueue(instance.GetResourceQueue().GetPhaseQueue())

		decrypt.Measure(measure.TagReceiveOnReception)

		if !queued {
			return errors.Errorf("Error on first node SharePhasePiece " +
				"comm, should be able to queue decrypt phase")
		}
		err = PostPhase(decrypt, blankBatch)

		if err != nil {
			return errors.Errorf("Error on first node SharePhasePiece "+
				"comm, should be able to post to decrypt phase: %+v", err)
		}
	}
	return nil
}

// isAlreadyParticipant is a helper function checking if we have sent our piece
// Returns true if ourId is in the participants list provided,
// otherwise returns false
func isAlreadyParticipant(participants [][]byte, ourID []byte) bool {
	for _, participantId := range participants {
		// If we already exist withing the participants list, return true
		if bytes.Equal(participantId, ourID) {
			return true
		}
	}

	return false
}

// checkFinalKeys is a helper function which checks if all keys in the
// keys slice are all identical and thus a valid key negotiation occurred
func checkFinalKeys(keys []*cyclic.Int) error {
	for i := 0; i < len(keys)-1; i++ {
		thisKey := keys[i]
		nextKey := keys[i+1]
		if thisKey.Cmp(nextKey) != 0 {
			return errors.Errorf("Keys from sub-rounds were not identical")
		}
	}
	return nil
}

// checkSignatures is a helper function which goes through every message
// sent in postSharePhase, verifying the signatures from each node
func checkSignatures(r *round.Round) error {
	topology := r.GetTopology()
	for i := 0; i < topology.Len(); i++ {
		nodeInfo := topology.GetHostAtIndex(i)
		msgs := r.GetPieceMessagesByNode(nodeInfo.GetId())
		// Check every message from every node in team
		for _, msg := range msgs {
			// Check if the signature for this message is valid
			if err := signature.Verify(msg, nodeInfo.GetPubKey()); err != nil {
				return errors.Errorf("Could not verify signature for node [%s]: %v",
					nodeInfo.GetId(), err)
			}
		}

	}
	return nil
}
