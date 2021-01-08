///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Contains sending functions for StartSharePhase and SharePhasePiece

package io

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/mixmessages"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/xx_network/comms/signature"
	"gitlab.com/xx_network/primitives/id"
	"sync"
)

// Triggers the multi-party communication in which generation of the
// round's Diffie-Hellman key will be generated. This triggers all other
// nodes in the team to start generating and sending out shares.
func TransmitStartSharePhase(roundID id.Round, serverInstance phase.GenericInstance, getChunk phase.GetChunk, getMessage phase.GetMessage) error {
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

	// Send the trigger to everyone in the round
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
// transmits it to all other nodes in the team. If theirPiece is non-nil,
// then our piece generation is a response to receiving a shared piece
// from a teammate. In this case, we create our piece off of their piece
// by exponentiating on their share with our roundKey and sent that to the team.
// If theirPiece is nil, then we are generating our piece for the first time,
// exponentiating off of the group itself and our round key, sending that that
// to the team.
func TransmitPhaseShare(instance *internal.Instance, r *round.Round,
	theirPiece *pb.SharePiece) error {

	grp := instance.GetConsensus().GetCmixGroup()

	// Build the message to be sent to all other nodes
	ourPiece := generateShare(theirPiece, grp,
		r, instance.GetID())

	// Sign our message for other nodes to verify
	err := signature.Sign(ourPiece, instance.GetPrivKey())
	if err != nil {
		return errors.Errorf("Could not sign message: %s", err)
	}

	// Send the shared theirPiece to all other nodes (including ourselves)
	topology := r.GetTopology()
	var wg sync.WaitGroup
	errChan := make(chan error, topology.Len())
	for i := 0; i < topology.Len(); i++ {
		wg.Add(1)
		localIndex := i

		go func() {
			h := topology.GetHostAtIndex(localIndex)
			ack, err := instance.GetNetwork().SendSharePhase(h, ourPiece)
			if err != nil {
				errChan <- errors.Errorf("Could not send to node [%s]: %v", h.GetId(), err)
			}

			if ack != nil && ack.Error != "" {
				errChan <- errors.Errorf("Remote Server Error: %s", ack.Error)
			}

			wg.Done()
		}()
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
// sent to all nodes in the round. If this is a response to a received
// share, we exponentiate on that share. If this is a response to a
// StartSharePhase (theirPiece is nil), we exponentiate the group on our key
func generateShare(theirPiece *pb.SharePiece, grp *cyclic.Group,
	rnd *round.Round, ourId *id.ID) *pb.SharePiece {

	roundKey := rnd.GetBuffer().Z
	// Checks if we are the first participant to generate a share
	if theirPiece == nil {
		// If first, generate our piece off of grp and our key
		// and add ourselves to the participant list
		newPiece := grp.ExpG(roundKey, grp.NewInt(1))
		participants := [][]byte{ourId.Bytes()}
		return &pb.SharePiece{
			Piece:        newPiece.Bytes(),
			Participants: participants,
			RoundID:      uint64(rnd.GetID()),
		}
	}

	// If we are not the first, raise the existing piece by our key
	// and add ourselves to the participant list
	oldPiece := grp.NewIntFromBytes(theirPiece.Piece)
	newPiece := grp.Exp(oldPiece, roundKey, grp.NewInt(1))
	participants := theirPiece.Participants
	participants = append(participants, ourId.Bytes())

	return &pb.SharePiece{
		Piece:        newPiece.Bytes(),
		Participants: participants,
		RoundID:      uint64(rnd.GetID()),
	}
}
