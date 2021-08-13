///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package io

// receiveFinishRealtime.go contains handler for finishRealtime.

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"git.xx.network/elixxir/comms/mixmessages"
	"git.xx.network/elixxir/primitives/current"
	"git.xx.network/elixxir/server/internal"
	"git.xx.network/elixxir/server/internal/measure"
	"git.xx.network/elixxir/server/internal/phase"
	"git.xx.network/xx_network/comms/connect"
	"git.xx.network/xx_network/primitives/id"
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
		roundErr := errors.Errorf("[%v]: Error on reception of "+
			"FinishRealtime comm, should be able to return: \n %+v",
			instance, err)
		return roundErr
	}
	p.Measure(measure.TagVerification)
	go func() {
		p.UpdateFinalStates()
		/*if !r.GetTopology().IsFirstNode(instance.GetID()) {
			// Disconnect from all hosts that are not "you"
			for i := 0; i < r.GetTopology().Len(); i++ {
				if !r.GetTopology().GetNodeAtIndex(i).Cmp(instance.GetID()) {
					r.GetTopology().GetHostAtIndex(i).Disconnect()
				}
			}
		}*/
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

			// Disconnect from all hosts that are not "you"
			//
			// In theory, this can run after the node gets a new round. If that
			// round includes a repeat node (which is very rare) it may
			// disconnect from a node which is in use. In such a case, the next
			// operation will reconnect. Given how unlikely this event is and
			// the auto recovery, we do not really care.
			/*for i := 0; i < r.GetTopology().Len(); i++ {
				if !r.GetTopology().GetNodeAtIndex(i).Cmp(instance.GetID()) {
					r.GetTopology().GetHostAtIndex(i).Disconnect()
				}
			}*/
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
