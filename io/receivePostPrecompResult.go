////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package io

// receivePostPrecompResult.go contains the handler for PostPrecompResult comm

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// ReceivePostPrecompResult handles the state checks and edge checks of
// receiving the result of the precomputation
func ReceivePostPrecompResult(instance *internal.Instance, roundID uint64,
	numslots uint32, auth *connect.Auth) error {

	curActivity, err := instance.GetStateMachine().WaitFor(250*time.Millisecond, current.PRECOMPUTING)
	if err != nil {
		return errors.WithMessagef(err, errFailedToWait, current.PRECOMPUTING.String())
	}
	if curActivity != current.PRECOMPUTING {
		return errors.Errorf(errCouldNotWait, current.PRECOMPUTING.String())
	}

	rm := instance.GetRoundManager()
	rid := id.Round(roundID)
	r, err := rm.GetRound(rid)
	if err != nil {
		return errors.WithMessagef(err, "Failed to retrieve round %+v", roundID)
	}

	// Check for proper authentication and expected sender
	expectedID := r.GetTopology().GetLastNode()
	senderID := auth.Sender.GetId()
	if !auth.IsAuthenticated || !senderID.Cmp(expectedID) {
		jww.INFO.Printf("[%v]: RID %d PostPrecompResult failed auth "+
			"(expected ID: %s, received ID: %s, auth: %v)",
			instance, roundID, expectedID, auth.Sender.GetId(),
			auth.IsAuthenticated)
		return connect.AuthError(auth.Sender.GetId())
	}

	jww.INFO.Printf("[%v]: RID %d PostPrecompResult START", instance,
		roundID)

	tag := phase.PrecompReveal.String() + "Verification"
	r, p, err := rm.HandleIncomingComm(id.Round(roundID), tag)
	if err != nil {
		roundErr := errors.Errorf("[%v]: Error on reception of "+
			"PostPrecompResult comm, should be able to return: \n %+v",
			instance, err)
		return roundErr
	}
	p.Measure(measure.TagVerification)
	if r.GetBatchSize() != numslots {
		roundErr := errors.Errorf("PostPrecompResult: The number of slots"+
			" we got (%d) wasn't equal to the number of slots in the "+
			"buffer (%d)", numslots, r.GetBatchSize())
		go func() {
			instance.ReportRoundFailure(roundErr, instance.GetID(), rid)
		}()
		return roundErr
	}

	// if the lastnode -> team test broadcast was not succesful
	// by now, fail the round
	if !r.PrecompBroadcastSuccess() {
		roundErr := errors.Errorf("Cannot handle completion of Premputation " +
			"because the last node test broadcat did not complete")
		go func() {
			instance.ReportRoundFailure(roundErr, instance.GetID(), rid)
		}()

		return roundErr
	}

	p.UpdateFinalStates()

	// Update the state in a gofunc
	go func() {
		ok, err := instance.GetStateMachine().Update(current.STANDBY)
		if err != nil {
			roundErr := errors.Errorf("Failed to transition to state STANDBY: %+v", err)
			instance.ReportRoundFailure(roundErr, instance.GetID(), rid)
		}
		if !ok {
			roundErr := errors.Errorf("Could not transition to state STANDBY")
			instance.ReportRoundFailure(roundErr, instance.GetID(), rid)
		}
	}()
	return nil
}
