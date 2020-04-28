////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package receivers

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/io"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/server/phase"
	"time"
)

// ReceivePostPrecompResult handles the state checks and edge checks of
// receiving the result of the precomputation
func ReceivePostPrecompResult(instance *server.Instance, roundID uint64,
	slots []*mixmessages.Slot, auth *connect.Auth) error {

	curActivity, err := instance.GetStateMachine().WaitFor(250*time.Millisecond, current.PRECOMPUTING)
	if err != nil {
		return errors.WithMessagef(err, errFailedToWait, current.PRECOMPUTING.String())
	}
	if curActivity != current.PRECOMPUTING {
		return errors.Errorf(errCouldNotWait, current.PRECOMPUTING.String())
	}

	rm := instance.GetRoundManager()
	r, err := rm.GetRound(id.Round(roundID))
	if err != nil {
		return errors.WithMessagef(err, "Failed to retrieve round %+v", roundID)
	}

	// Check for proper authentication and expected sender
	expectedID := r.GetTopology().GetLastNode().String()
	senderID := auth.Sender.GetId()
	if !auth.IsAuthenticated || senderID != expectedID {
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
	err = io.PostPrecompResult(r.GetBuffer(), instance.GetConsensus().GetCmixGroup(), slots)
	if err != nil {
		return errors.Wrapf(err,
			"Couldn't post precomp result for round %v", roundID)
	}
	p.UpdateFinalStates()

	// Update the state in a gofunc
	go func() {
		ok, err := instance.GetStateMachine().Update(current.STANDBY)
		if err != nil {
			roundErr := errors.Errorf("Failed to transition to state STANDBY: %+v", err)
			instance.ReportRoundFailure(roundErr, instance.GetID())
		}
		if !ok {
			roundErr := errors.Errorf("Could not transition to state STANDBY")
			instance.ReportRoundFailure(roundErr, instance.GetID())
		}
	}()
	return nil
}
