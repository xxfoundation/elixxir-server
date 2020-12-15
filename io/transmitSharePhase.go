///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Contains sending functions for StartSharePhase and SharePhaseRound

package io

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/xx_network/comms/signature"
	"gitlab.com/xx_network/primitives/id"
)

// Triggers the multi-party communication in which generation of the round's Diffie-Helman key
// will be generated
func TransmitStartSharePhase(roundID id.Round, serverInstance phase.GenericInstance) error {
	// Cast the instance into the proper internal type
	instance, ok := serverInstance.(*internal.Instance)
	if !ok {
		return errors.Errorf("Invalid server instance passed in")
	}

	//get the round so you can get its batch size
	r, err := instance.GetRoundManager().GetRound(roundID)
	if err != nil {
		return errors.Errorf("Received completed batch for round %v that doesn't exist: %s", roundID, err)
	}

	topology := r.GetTopology()

	ri := &mixmessages.RoundInfo{
		ID: uint64(roundID),
	}

	// Attempt to sign the round info being passed to the next round
	if err = signature.Sign(ri, instance.GetPrivKey()); err != nil {
		jww.FATAL.Panicf("Could not start share phase: "+
			"Failed to sign round info for round [%d]: %s ", roundID, err)
	}

	// Send the trigger to everyone in the round
	for i := 0; i < topology.Len(); i++ {
		h := topology.GetHostAtIndex(i)
		ack, err := instance.GetNetwork().SendStartSharePhase(h, ri)
		if ack != nil && ack.Error != "" || err != nil {
			err = errors.Errorf("Remote Server Error: %s", ack.Error)
		}
	}

	return err
}

// TransmitPhaseShare is a helper function which generates our local shared piece
// of the round key. It also sends the message to all other nodes in the team
func TransmitPhaseShare(instance *internal.Instance, r *round.Round,
	theirPiece *pb.SharePiece) error {

	var newPiece *cyclic.Int
	participants := make([][]byte, 0)
	generator := instance.GetConsensus().GetCmixGroup()
	roundKey := r.GetBuffer().Z

	// Checks if we are the first participant to generate a share
	if theirPiece == nil {
		// If first, generate our piece off of generator and our key
		// and add ourselves to the participant list
		newPiece = generator.ExpG(roundKey, generator.NewInt(1))
		participants = [][]byte{instance.GetID().Bytes()}
	} else {
		// If we are not the first, raise the existing piece by our key
		// and add ourselves to the participant list
		oldPiece := generator.NewIntFromBytes(theirPiece.Piece)
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

	// Send the shared theirPiece to all other nodes (including ourselves)
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
