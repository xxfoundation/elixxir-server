///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Contains sending functions for ReceiveStartSharePhase and ReceiveSharePhasePiece

package io

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/mixmessages"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/xx_network/comms/signature"
	"gitlab.com/xx_network/primitives/id"
	"sync"
)

// Triggers the multi-party communication in which generation of the
// round's Diffie-Hellman key will be generated. This triggers all other
// nodes in the team to start generating and sending out shares.
func TransmitStartSharePhase(roundID id.Round, instance *internal.Instance) error {

	// Get the round so you can get its batch size
	r, err := instance.GetRoundManager().GetRound(roundID)
	if err != nil {
		return errors.Errorf("Received completed batch for round %v that doesn't exist: %s", roundID, err)
	}

	topology := r.GetTopology()

	ri := &mixmessages.RoundInfo{
		ID: uint64(roundID),
	}

	// Send the final key to everyone in the round
	errChan := make(chan error, topology.Len())
	var wg sync.WaitGroup
	for i := 0; i < topology.Len(); i++ {
		wg.Add(1)
		go func(localIndex int) {
			h := topology.GetHostAtIndex(localIndex)
			ack, err := instance.GetNetwork().SendStartSharePhase(h, ri)
			if err != nil {
				errChan <- errors.Wrapf(err, "")
			}

			if ack != nil && ack.Error != "" {
				errChan <- errors.Errorf("Remote Server Error: %s", ack.Error)
			}
			wg.Done()
		}(i)
	}
	// Wait for all responses
	wg.Wait()

	// Return all node comms or ack errors if any
	// as a single error message
	var errs error
	for len(errChan) > 0 {
		err := <-errChan
		if errs != nil {
			errs = errors.Wrap(errs, err.Error())
		} else {
			errs = err
		}
	}

	return errs
}

// TransmitPhaseShare is send function which generates our shared piece and
// transmits it to our neighboring node. We exponentiate the piece with our
// private round key.
func TransmitPhaseShare(instance *internal.Instance, r *round.Round,
	theirPiece *pb.SharePiece) error {

	grp := instance.GetConsensus().GetCmixGroup()

	// Build the message to be sent to all other nodes
	ourPiece, err := generateShare(theirPiece, grp,
		r, instance)

	// Pull recipient to send for the next node
	topology := r.GetTopology()
	recipientID := topology.GetNextNode(instance.GetID())
	nextNodeIndex := topology.GetNodeLocation(recipientID)
	recipient := topology.GetHostAtIndex(nextNodeIndex)

	// Send share to next node
	ack, err := instance.GetNetwork().SendSharePhase(recipient, ourPiece)
	if err != nil {
		return errors.Errorf("Could not send to node [%s]: %v",
			recipient.GetId(), err)
	}
	if ack != nil && ack.Error != "" {
		return errors.Errorf("Remote Server Error: %s", ack.Error)
	}

	return nil
}

// TransmitFinalShare is a function which sends out the final key receive
// all other nodes in the team.
func TransmitFinalShare(instance *internal.Instance, r *round.Round,
	finalPiece *pb.SharePiece) error {

	topology := r.GetTopology()

	// Send the trigger to everyone in the round
	errChan := make(chan error, topology.Len())
	var wg sync.WaitGroup
	for i := 0; i < topology.Len(); i++ {
		wg.Add(1)
		go func(localIndex int) {
			// Send to every node other than ourself
			h := topology.GetHostAtIndex(localIndex)
			ack, err := instance.GetNetwork().SendFinalKey(h, finalPiece)
			if err != nil {
				errChan <- errors.Wrapf(err, "")
			}

			if ack != nil && ack.Error != "" {
				errChan <- errors.Errorf("Remote Server Error: %s", ack.Error)

			}
			wg.Done()
		}(i)
	}
	// Wait for all responses
	wg.Wait()

	// Return all node comms or ack errors if any
	// as a single error message
	var errs error
	for len(errChan) > 0 {
		err := <-errChan
		if errs != nil {
			errs = errors.Wrap(errs, err.Error())
		} else {
			errs = err
		}
	}

	return errs

}

// generateShare is a helper function which generates a key share to be
// sent to the next node in the round. We exponentiate the received piece
// with our round's private key
func generateShare(theirPiece *pb.SharePiece, grp *cyclic.Group,
	rnd *round.Round, instance *internal.Instance) (*pb.SharePiece, error) {

	roundPrivKey := rnd.GetBuffer().Z

	// Raise the existing piece by our key
	// and add ourselves to the participant list
	oldPiece := grp.NewIntFromBytes(theirPiece.Piece)
	newPiece := grp.Exp(oldPiece, roundPrivKey, grp.NewInt(1))
	participants := theirPiece.Participants
	participants = append(participants, instance.GetID().Bytes())

	ourPiece := &pb.SharePiece{
		Piece:        newPiece.Bytes(),
		Participants: participants,
		RoundID:      uint64(rnd.GetID()),
	}

	// Sign our message for other nodes to verify
	if err := signature.Sign(ourPiece, instance.GetPrivKey()); err != nil {
		return nil, errors.Errorf("Could not sign message: %s", err)
	}

	return ourPiece, nil
}
