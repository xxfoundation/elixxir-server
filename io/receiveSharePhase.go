///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package io

// Contains handlers for ReceiveStartSharePhase and ReceiveSharePhasePiece

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
	"gitlab.com/elixxir/server/internal/state"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/signature"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// ReceiveStartSharePhase is a reception handler for TransmitStartSharePhase.
// It does basic checks of the validity of the message and the sender.
// After checks are complete, it generates and transmits its own share by
// calling TransmitPhaseShare
func ReceiveStartSharePhase(ri *pb.RoundInfo, auth *connect.Auth,
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
	senderId := auth.Sender.GetId()
	if !auth.IsAuthenticated || topology.GetNodeLocation(senderId) == -1 {
		jww.WARN.Printf("Error on ReceiveStartSharePhase: "+
			"Attempted communication by %+v has not been authenticated: %s",
			auth.Sender, auth.Reason)
		return errors.WithMessage(connect.AuthError(auth.Sender.GetId()), auth.Reason)
	}

	// Check if we are in the proper stage
	tag := phase.PrecompShare.String()
	_, p, err := rm.HandleIncomingComm(roundID, tag)
	if err != nil {
		return errors.Errorf("[%v]: Error on reception of "+
			"ReceiveStartSharePhase comm, should be able to return: \n %+v",
			instance, err)
	}
	p.Measure(measure.TagReceiveOnReception)

	// Update our internal phase machine to started
	ok, err := instance.GetPhaseShareMachine().Update(state.STARTED)
	if err != nil {
		jww.ERROR.Printf("Failed to transition to state STARTED: %+v", err)
	}
	if !ok {
		jww.ERROR.Printf("Could not transition to state STARTED")
	}

	// If first, generate our piece off of grp and our key
	// and add ourselves to the participant list
	grp := instance.GetConsensus().GetCmixGroup()
	msg := &pb.SharePiece{
		Piece:        grp.GetG().Bytes(),
		Participants: make([][]byte, 0),
		RoundID:      uint64(r.GetID()),
	}

	// Generate and sign the  message to be shared with the team
	if err = TransmitPhaseShare(instance, r, msg); err != nil {
		jww.FATAL.Panicf("Error on ReceiveStartSharePhase: "+
			"Could not send our shared piece of the key: %v", err)
	}

	return nil
}

// ReceiveSharePhasePiece is a reception handler for receiving a key generation share.
// It does basic validity checks on the share and the sender. If our node is not
// in the list of participants of the message, we generate a share off of the
// received piece and send that share all other teammates.
// We also update our state for this received message
func ReceiveSharePhasePiece(piece *pb.SharePiece, auth *connect.Auth,
	instance *internal.Instance) error {

	// Nil checks on received message
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

	// Check the local phase state machine for proper state
	curStatus, err := instance.GetPhaseShareMachine().WaitFor(100*time.Millisecond, state.STARTED)
	if err != nil {
		return errors.WithMessagef(err, errFailedToWait, state.STARTED.String())
	}
	if curStatus != state.STARTED {
		return errors.Errorf(errCouldNotWait, state.STARTED.String())

	}

	// Get round from round manager
	roundID := id.Round(piece.RoundID)
	rm := instance.GetRoundManager()
	r, err := rm.GetRound(roundID)
	if err != nil {
		return errors.WithMessage(err, "Failed to get round")
	}

	// Check for proper authentication and if the sender is in the round
	senderId := auth.Sender.GetId()
	topology := r.GetTopology()
	prevNode := topology.GetPrevNode(instance.GetID())

	if !auth.IsAuthenticated || !prevNode.Cmp(senderId) {
		jww.WARN.Printf("ReceiveSharePhasePiece Error: "+
			"Attempted communication by %+v has not been authenticated: %s",
			auth.Sender, auth.Reason)
		return errors.WithMessage(connect.AuthError(auth.Sender.GetId()), auth.Reason)
	}

	participants := piece.Participants
	if isAlreadyParticipant(participants, instance.GetID().Bytes()) {
		// If we are a participant, then our piece has come full circle.
		// send out final key to all members
		if err := TransmitFinalShare(instance, r, piece); err != nil {
			jww.FATAL.Panicf("Error on ReceiveSharePhasePiece: "+
				"Could not send our final key: %s", err)
		}
	} else {
		// If not a participant, send message to neighboring node
		if err = TransmitPhaseShare(instance, r, piece); err != nil {
			jww.FATAL.Panicf("Error on ReceiveSharePhasePiece: "+
				"Could not send our shared piece of the key: %s", err)
		}
	}

	return nil
}

// ReceiveFinalKey is a reception handler for a node transmitting their
// team-generated final key. This node checks for validity of the message
// and its own state. Further checks the validity of the key received,
// and updates it's own state upon validation. If all final keys have been
// received, this node finalizes key generation and moves it's phase forward
func ReceiveFinalKey(piece *pb.SharePiece, auth *connect.Auth,
	instance *internal.Instance) error {

	// Nil checks on received message
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

	// Check the local phase state machine for proper state
	curStatus, err := instance.GetPhaseShareMachine().WaitFor(100*time.Millisecond, state.STARTED)
	if err != nil {
		return errors.WithMessagef(err, errFailedToWait, state.STARTED.String())
	}
	if curStatus != state.STARTED {
		return errors.Errorf(errCouldNotWait, state.STARTED.String())
	}

	// Get round from round manager
	roundID := id.Round(piece.RoundID)
	rm := instance.GetRoundManager()
	r, err := rm.GetRound(roundID)
	if err != nil {
		return errors.WithMessage(err, "Failed to get round")
	}

	// Check for proper authentication and if the
	// sender is the leader of the round
	topology := r.GetTopology()
	senderId := auth.Sender.GetId()
	if !auth.IsAuthenticated || topology.GetNodeLocation(senderId) == -1 {
		jww.WARN.Printf("Error on ReceiveStartSharePhase: "+
			"Attempted communication by %+v has not been authenticated: %s",
			auth.Sender, auth.Reason)
		return errors.WithMessage(connect.AuthError(auth.Sender.GetId()), auth.Reason)
	}

	// Check the validity
	if err = updateFinalKeys(piece, r, senderId, instance); err != nil {
		jww.FATAL.Panicf("Could not verify final key received: %v", err)

	}
	return nil
}

// updateFinalKeys is a helper function for ReceiveFinalKey.
// It checks the validity of the supposed final key. If valid,
// it adds it to the state. If we have received final keys from all
// nodes, final key generation is initiated.
func updateFinalKeys(piece *pb.SharePiece, r *round.Round, originID *id.ID,
	instance *internal.Instance) error {
	// Pull the participants and teamsize information
	participants := piece.Participants
	teamSize := r.GetTopology().Len()

	// Check if this shared piece has gone through all nodes
	if len(participants) != teamSize {
		return errors.Errorf("Potential final key was not " +
			"received by all members of the team")
	}

	// Add the key with all shares to our state
	grp := instance.GetConsensus().GetCmixGroup()
	roundKey := grp.NewIntFromBytes(piece.Piece)
	// Check that this node hasn't already sent a final key
	if err := r.AddFinalShareMessage(piece, originID); err != nil {
		return errors.Errorf("Failed to add final key "+
			"from [%s]: %v", originID, err)
	}
	finalKeys := r.UpdateFinalKeys(roundKey)

	// If we have received final keys from every node
	if len(finalKeys) == teamSize {
		return finalizeKeyGeneration(instance, r, finalKeys)
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

	ok, err := instance.GetPhaseShareMachine().Update(state.ENDED)
	if err != nil {
		return errors.Errorf("Failed to transition to state ENDED: %+v", err)
	}
	if !ok {
		return errors.Errorf("Could not transition to state ENDED")
	}
	// Once done, transition from precompShare to precompDecrypt
	if err := transitionToPrecompDecrypt(instance, r); err != nil {
		return errors.Errorf("Could not transition to precompDecrypt: %s", err)
	}

	return nil
}

// transitionToPrecompDecrypt is a helper function which handles the business
// logic of transitioning from precompShare to precompDecrypt
func transitionToPrecompDecrypt(instance *internal.Instance, r *round.Round) error {

	rm := instance.GetRoundManager()
	topology := r.GetTopology()
	roundID := r.GetID()
	p, err := rm.GetPhase(roundID, int32(phase.PrecompShare))
	if err != nil {
		return errors.Errorf("Could not pull phase: %v", err)
	}

	jww.INFO.Printf("[%v]: RID %d ReceiveSharePhasePiece PK is: %s",
		instance, roundID, r.GetBuffer().CypherPublicKey.Text(16))

	p.UpdateFinalStates()

	jww.INFO.Printf("[%v]: RID %d ReceiveSharePhasePiece END", instance,
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
			return errors.Errorf("Error on first node ReceiveSharePhasePiece "+
				"comm, should be able to get decrypt phase: %+v", err)
		}

		jww.INFO.Printf("[%v]: RID %d ReceiveSharePhasePiece FIRST NODE START PHASE \"%s\"", instance,
			roundID, decrypt.GetType())

		queued :=
			decrypt.AttemptToQueue(instance.GetResourceQueue().GetPhaseQueue())

		decrypt.Measure(measure.TagReceiveOnReception)

		if !queued {
			return errors.Errorf("Error on first node ReceiveSharePhasePiece " +
				"comm, should be able to queue decrypt phase")
		}
		err = PostPhase(decrypt, blankBatch)

		if err != nil {
			return errors.Errorf("Error on first node ReceiveSharePhasePiece "+
				"comm, should be able to post to decrypt phase: %+v", err)
		}
	}

	ok, err := instance.GetPhaseShareMachine().Reset()
	if err != nil {
		return errors.Errorf("Failed to transition to state ENDED: %+v", err)
	}
	if !ok {
		return errors.Errorf("Could not transition to state ENDED")
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
			return errors.Errorf("Keys generated from phase share were not identical")
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
		msg := r.GetPieceMessagesByNode(nodeInfo.GetId())
		// Check every message from every node in team
		// Check if the signature for this message is valid
		if nodeInfo.GetPubKey() != nil {
			if err := signature.Verify(msg, nodeInfo.GetPubKey()); err != nil {
				return errors.Errorf("Could not verify signature for node [%s]: %v",
					nodeInfo.GetId(), err)
			}
		}

	}
	return nil
}
