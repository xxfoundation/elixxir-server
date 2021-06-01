///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package io

// receivePostPhase.go contains the handler for server <-> server postPhase comm

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// ReceivePostPhase handles the state checks and edge checks of receiving a
// phase operation
func ReceivePostPhase(batch *mixmessages.Batch, instance *internal.Instance, auth *connect.Auth) error {

	// HACK HACK HACK
	// in the event not started hasn't finished, this waits for ti to finish
	// or is ignored otherwise
	_, _ = instance.GetStateMachine().WaitFor(5*time.Second, current.WAITING)

	// Wait until acceptable state to start post phase
	curActivity, err := instance.GetStateMachine().WaitFor(3*time.Second, current.PRECOMPUTING, current.REALTIME)
	if err != nil {
		return errors.WithMessagef(err, errFailedToWait, "from: "+phase.Type(batch.FromPhase).String())
	}

	nodeID := instance.GetID()
	roundID := id.Round(batch.Round.ID)
	phaseTy := phase.Type(batch.FromPhase).String()

	rm := instance.GetRoundManager()
	r, err := rm.GetRound(roundID)
	if err != nil {
		return errors.WithMessagef(err, "Failed to get round %d", roundID)
	}

	topology := r.GetTopology()
	prevNodeID := topology.GetPrevNode(nodeID)

	// Check for proper authentication and if the sender
	// is the previous node in the circuit
	if !auth.IsAuthenticated || !prevNodeID.Cmp(auth.Sender.GetId()) {
		jww.WARN.Printf("Error on PostPhase: "+
			"Attempted communication by %+v has not been authenticated: %s", auth.Sender, auth.Reason)
		return errors.WithMessage(connect.AuthError(auth.Sender.GetId()), auth.Reason)
	}

	// Waiting for correct phase
	ptype := r.GetCurrentPhaseType()
	toWait := shouldWait(ptype)
	if toWait == current.ERROR {
		return errors.Errorf("Phase %+s has not associated node activity", ptype)
	} else if toWait != curActivity {
		return errors.Errorf("System in wrong state. Expected state: %s\nActual state: %s\n Current phase: %s",
			toWait, curActivity, phaseTy)
	}

	//Check if the operation can be done and get the correct phase if it can
	_, p, err := rm.HandleIncomingComm(roundID, phaseTy)
	if err != nil {
		roundErr := errors.Errorf("[%v]: Error on reception of "+
			"PostPhase comm, should be able to return: \n %+v",
			instance, err)
		return roundErr
	}
	p.Measure(measure.TagReceiveOnReception)

	jww.INFO.Printf("[%v]: RID %d PostPhase FROM \"%s\" FOR \"%s\" RECEIVE/START", instance,
		roundID, phaseTy, p.GetType())
	//if the phase has an alternate action, use that
	if has, alternate := p.GetAlternate(); has {
		go alternate()
		return nil
	}

	//queue the phase to be operated on if it is not queued yet
	p.AttemptToQueue(instance.GetResourceQueue().GetPhaseQueue())

	batch.FromPhase = int32(p.GetType())
	//send the data to the phase
	err = PostPhase(p, batch)

	if err != nil {
		roundErr := errors.Errorf("Error on PostPhase comm, should be"+
			" able to return: %+v", err)
		return roundErr
	}
	return nil
}

// ReceiveStreamPostPhase handles the state checks and edge checks of
// receiving a phase operation
func ReceiveStreamPostPhase(streamServer mixmessages.Node_StreamPostPhaseServer,
	instance *internal.Instance, auth *connect.Auth) error {

	// Get batch info
	batchInfo, err := node.GetPostPhaseStreamHeader(streamServer)
	if err != nil {
		return errors.WithMessage(err, "Could not get post phase stream header")
	}
	roundID := id.Round(batchInfo.Round.ID)
	rm := instance.GetRoundManager()
	r, err := rm.GetRound(roundID)
	if err != nil {
		return errors.WithMessagef(err, "Failed to get round %d", roundID)
	}
	topology := r.GetTopology()

	// Check for proper authentication and expected sender
	nodeID := instance.GetID()
	prevNodeID := topology.GetPrevNode(nodeID)
	if !auth.IsAuthenticated || !prevNodeID.Cmp(auth.Sender.GetId()) {
		errMsg := errors.Errorf("[%v]: Reception of StreamPostPhase comm failed authentication: "+
			"(Expected ID: %+v, received id: %+v.\n Auth: %+v)", instance,
			prevNodeID, auth.Sender.GetId(), auth.IsAuthenticated)

		jww.ERROR.Println(errMsg)
		return errMsg

	}

	// Waiting for correct phase
	ptype := r.GetCurrentPhaseType()
	toWait := shouldWait(ptype)

	curActivity, err := instance.GetStateMachine().WaitFor(3*time.Second, toWait)
	if err != nil {
		return errors.WithMessagef(err, errFailedToWait, ptype)
	}

	if toWait == current.ERROR {
		return errors.Errorf("Phase %+s has not associated node activity", ptype)
	} else if toWait != curActivity {
		return errors.Errorf("System in wrong state. Expected state: %s\nActual state: %s\n Current phase: %s",
			toWait, curActivity, ptype)
	}

	phaseTy := phase.Type(batchInfo.FromPhase).String()

	// Check if the operation can be done and get the correct
	// phase if it can
	_, p, err := rm.HandleIncomingComm(roundID, phaseTy)
	if err != nil {
		roundErr := errors.Errorf("[%v]: Error on reception of "+
			"StreamPostPhase comm, should be able to return: \n %+v",
			instance, err)
		return roundErr
	}
	p.Measure(measure.TagReceiveOnReception)

	jww.INFO.Printf("[%v]: RID %d StreamPostPhase FROM \"%s\" TO \"%s\" RECEIVE/START", instance,
		roundID, phaseTy, p.GetType())

	//queue the phase to be operated on if it is not queued yet
	p.AttemptToQueue(instance.GetResourceQueue().GetPhaseQueue())

	start, end, strmErr := StreamPostPhase(p, batchInfo.BatchSize, streamServer)

	jww.INFO.Printf("\tbwLogging: Round %d, "+
		"received phase: %s, "+
		"from: %s, to: %s, "+
		"started: %v, "+
		"ended: %v, "+
		"duration: %d,",
		roundID, phaseTy,
		auth.Sender.GetId().String(), instance.GetID(),
		start, end, end.Sub(start).Milliseconds())

	return strmErr

}
