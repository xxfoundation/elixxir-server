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
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/phase"
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

//
func TransmitSharePhase() error {
	// todo: implement me
	return nil
}
