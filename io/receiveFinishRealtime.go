////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

// receiveFinishRealtime.go contains handler for finishRealtime.

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/phase"
	"time"
)

// ReceiveFinishRealtime handles the state checks and edge checks of
// receiving the signal that the realtime has completed
func ReceiveFinishRealtime(instance *internal.Instance, msg *mixmessages.RoundInfo,
	auth *connect.Auth) error {
	// Get round from round manager
	roundID := id.Round(msg.ID)
	rm := instance.GetRoundManager()
	r, err := rm.GetRound(roundID)
	if err != nil {
		return errors.WithMessage(err, "Failed to get round")
	}

	expectedID := r.GetTopology().GetLastNode()
	if !auth.IsAuthenticated || !auth.Sender.GetId().Cmp(expectedID) {
		jww.INFO.Printf("[%v]: RID %d FinishRealtime failed auth "+
			"(expected ID: %s, received ID: %s, auth: %v)",
			instance, roundID, expectedID, auth.Sender.GetId(),
			auth.IsAuthenticated)
		return connect.AuthError(auth.Sender.GetId())
	}

	curActivity, err := instance.GetStateMachine().WaitFor(250*time.Millisecond, current.REALTIME)
	if err != nil {
		return errors.WithMessagef(err, errFailedToWait, current.REALTIME.String())
	}
	if curActivity != current.REALTIME {
		return errors.Errorf(errCouldNotWait, current.REALTIME.String())
	}

	jww.INFO.Printf("[%v]: RID %d ReceiveFinishRealtime START",
		instance, roundID)

	tag := phase.RealPermute.String() + "Verification"
	r, p, err := rm.HandleIncomingComm(id.Round(roundID), tag)
	if err != nil {
		jww.FATAL.Panicf("[%v]: Error on reception of "+
			"FinishRealtime comm, should be able to return: \n %+v",
			instance, err)
	}
	p.Measure(measure.TagVerification)
	go func() {

		p.UpdateFinalStates()
		if !instance.GetKeepBuffers() {
			//Delete the round and its data from the manager
			//Delay so it can be used by post round hanlders
			go func() {
				jww.INFO.Printf("[%v]: RID %d ReceiveFinishRealtime CLEARING "+
					"CMIX BUFFERS", instance, roundID)

				time.Sleep(time.Duration(60) * time.Second)
				r.GetBuffer().Erase()
				rm.DeleteRound(roundID)
			}()

		} else {
			jww.WARN.Printf("[%v]: RID %d ReceiveFinishRealtime MEMORY "+
				"LEAK - Round buffers not purged ", instance,
				roundID)
		}
	}()

	jww.INFO.Printf("[%v]: RID %d ReceiveFinishRealtime END", instance,
		roundID)

	jww.INFO.Printf("[%v]: RID %d Round took %v seconds",
		instance, roundID, time.Now().Sub(r.GetTimeStart()))

	// If this is the first node, then start metrics collection.
	if r.GetTopology().IsFirstNode(instance.GetID()) {
		// The go routine that gathers all the metrics from all other nodes each
		// round and then saves them to a file.
		go func() {
			err = instance.GetDefinition().MetricsHandler(instance, roundID)
			if err != nil {
				jww.ERROR.Printf("Failure in posting metrics for round %d: %v",
					roundID, err)
			}
		}()
	}

	go func() {
		ok, err := instance.GetStateMachine().Update(current.COMPLETED)
		if err != nil {
			jww.ERROR.Printf(errors.WithMessagef(err, errFailedToUpdate, current.COMPLETED.String()).Error())
		}
		if !ok {
			jww.ERROR.Printf(errCouldNotUpdate, current.COMPLETED.String())
		}
	}()

	select {
	case r.GetMeasurementsReadyChan() <- struct{}{}:
	default:
	}

	return nil
}
