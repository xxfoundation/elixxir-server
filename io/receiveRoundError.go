///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package io

import (
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/signature"
	"gitlab.com/xx_network/primitives/id"
)

// ReceiveRoundError takes the round error message and checks if it's within the round
// If so then we transition to an error state. If not we ignore the error and send it
func ReceiveRoundError(msg *mixmessages.RoundError, auth *connect.Auth, instance *internal.Instance) error {

	// Get the round information from message. If the sent round information is
	// invalid, then we send back an error
	roundId := msg.GetId()
	rm := instance.GetRoundManager()
	r, err := rm.GetRound(id.Round(roundId))
	if err != nil {
		return errors.WithMessagef(err, "Failed to get round %d", roundId)
	}

	// Pull the erroring node id from message and check if it's valid
	badNodeId, err := id.Unmarshal(msg.GetNodeId())
	if err != nil {
		return errors.Errorf("Received unrecognizable node id: %v",
			err.Error())
	}

	// Build topology from round information
	topology := r.GetTopology()

	// Attempt to pull the node from the topology
	nodeIndex := topology.GetNodeLocation(badNodeId)

	// Check for proper authentication and if the sender is in the circuit
	// a -1 value indicates a non existant node in the topology
	if !auth.IsAuthenticated || nodeIndex == -1 {
		jww.INFO.Printf("[%v]: RID %d RoundError failed auth "+
			"(expected ID: %s, received ID: %s, auth: %v)",
			instance, roundId, badNodeId, auth.Sender.GetId(),
			auth.IsAuthenticated)
		return connect.AuthError(auth.Sender.GetId())
	}

	//check the signature on the round error is valid
	err = signature.VerifyRsa(msg, auth.Sender.GetPubKey())
	if err != nil {
		jww.WARN.Printf("Received an error for round %v from node %s "+
			"that could not be authenticated: %s, %+v", r.GetID(),
			auth.Sender.GetId(), err, msg)
		return errors.WithMessage(err, "could not verify round error")
	}

	// do edge checking to make sure the round is still ongoing, reject if it is
	// not an in progress round
	phaseState := r.GetCurrentPhase()

	if r.GetCurrentPhase().GetType() == phase.Complete ||
		r.GetCurrentPhase().GetType() == phase.PhaseError {
		jww.WARN.Printf("Received an error for round %v from node %s "+
			"when round is already complete: %s", r.GetID(),
			auth.Sender.GetId(), phaseState)
		return errors.New("Cannot process error associated with inactive round")
	}

	jww.ERROR.Printf("ReceiveRoundError received error from [%v]: %+v. Transitioning to ERROR...",
		badNodeId, msg.Error)

	//report the error in a seperate thread so this will return to the originator
	go func() {
		time.Sleep(100 * time.Millisecond)
		instance.ReportRemoteFailure(msg)
	}()

	return nil
}
