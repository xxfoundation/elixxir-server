///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package io

// Contains handlers for StartSharePhase and SharePhaseRound

import (
	"bytes"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/signature"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// Server -> Server initiating multi-party round DH key generation
// todo: better docstring explaining reception lgoic
func StartSharePhase(ri *pb.RoundInfo, instance *internal.Instance, auth *connect.Auth) error {

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

	topology := r.GetTopology()

	// Check for proper authentication
	// and if the sender is the leader of the round
	if !auth.IsAuthenticated || !topology.IsFirstNode(auth.Sender.GetId()) {
		jww.WARN.Printf("Error on PostPhase: "+
			"Attempted communication by %+v has not been authenticated: %s", auth.Sender, auth.Reason)
		return errors.WithMessage(connect.AuthError(auth.Sender.GetId()), auth.Reason)
	}

	// Verify signature of the received shared piece
	err = signature.Verify(ri, auth.Sender.GetPubKey())
	if err != nil {
		return errors.Errorf("Failed to verify signature from [%s]: %v", auth.Sender, err)
	}

	// Generate and sign the  message
	err = sendOurShare(instance, r, nil)
	if err != nil {
		jww.FATAL.Panicf("Error on StartSharePhase: "+
			"Could not send our shared piece of the key: %v", err)
	}

	return nil
}

// Server -> Server passing state of multi-party round DH key generation
// todo: better docstring explaining reception lgoic
func SharePhaseRound(piece *pb.SharePiece, auth *connect.Auth, instance *internal.Instance) error {
	// todo: add a phase lock?

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

	topology := r.GetTopology()
	participants := piece.Participants

	// Check for proper authentication and if the sender is in the round
	if !auth.IsAuthenticated || topology.GetNodeLocation(auth.Sender.GetId()) == -1 {
		jww.WARN.Printf("Error on SharePhaseRound: "+
			"Attempted communication by %+v has not been authenticated: %s",
			auth.Sender, auth.Reason)
		return errors.WithMessage(connect.AuthError(auth.Sender.GetId()), auth.Reason)
	}

	r.AddPieceMessage(piece, auth.Sender.GetId())

	// If we are aren't already a participant, we must send our share
	if !isAlreadyParticipant(participants, instance.GetID().Bytes()) {
		// Develop and send our share
		err = sendOurShare(instance, r, piece)
		if err != nil {
			jww.FATAL.Panicf("Error on StartSharePhase: "+
				"Could not send our shared piece of the key: %v", err)
		}

	}

	// Handle the state of our round
	handleRoundState(instance, r, piece)

	return nil
}

// sendOurShare is a helper function which generates our local shared piece
// of the round key. It also sends the message to all other nodes in the team
func sendOurShare(instance *internal.Instance, r *round.Round,
	piece *pb.SharePiece) error {

	var newPiece *cyclic.Int
	participants := make([][]byte, 0)
	generator := instance.GetConsensus().GetCmixGroup()
	roundKey := r.GetBuffer().Z

	// Checks if we are the first participant to generate a share
	if piece == nil {
		// If first, generate piece off of generator and our key
		// and add ourselves to the participant list
		newPiece = generator.ExpG(roundKey, generator.NewInt(1))
		participants = [][]byte{instance.GetID().Bytes()}
	} else {
		// If we are not the first, raise the existing piece by our key
		// and add ourselves to the participant list
		oldPiece := generator.NewIntFromBytes(piece.Piece)
		newPiece = generator.ExpG(oldPiece, roundKey)
		participants = append(participants, instance.GetID().Bytes())
	}

	// Build the message to be sent to all other nodes
	ourPiece := &pb.SharePiece{
		Piece:        newPiece.Bytes(),
		Participants: participants,
		RoundID:      uint64(r.GetID()),
	}

	// Sign our message for other nodes to verify
	err := signature.Sign(ourPiece, instance.GetPrivKey())
	if err != nil {
		return errors.Errorf("Could not sign message: %s", err)
	}

	r.AddPieceMessage(ourPiece)

	// Send the shared piece to all other nodes
	topology := r.GetTopology()
	for i := 0; i < topology.Len(); i++ {
		h := topology.GetHostAtIndex(i)
		_, err = instance.GetNetwork().SendSharePhase(h, ourPiece)
		if err != nil {
			return errors.Errorf("Could not send to node [%s]: %v", h.GetId(), err)
		}

	}

	return nil
}

func handleRoundState(instance *internal.Instance, r *round.Round, piece *pb.SharePiece) {
	// Once all checks are complete, increment the number of shares received
	newAmountOfShares := instance.IncrementShares()

	// Check if we are done with this round of sharing
	teamSize := uint32(r.GetTopology().Len())
	numberOfRounds := int(teamSize * teamSize)
	if newAmountOfShares == teamSize {
		finalKeys := r.GetFinalKeys()
		// Check if all rounds have been completed
		if numberOfRounds == len(finalKeys) {

		}
		// Append key to list of final keys
		// Initiate new round
		//
	}

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
