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
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/io"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/server/phase"
	"time"
)

// ReceivePostPhase handles the state checks and edge checks of receiving a
// phase operation
func ReceivePostPhase(batch *mixmessages.Batch, instance *server.Instance, auth *connect.Auth) error {

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
	if !auth.IsAuthenticated || prevNodeID.String() != auth.Sender.GetId() {
		jww.WARN.Printf("Error on PostPhase: "+
			"Attempted communication by %+v has not been authenticated", auth.Sender)
		return connect.AuthError(auth.Sender.GetId())
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
		jww.FATAL.Panicf("[%v]: Error on reception of "+
			"PostPhase comm, should be able to return: \n %+v",
			instance, err)
	}
	p.Measure(measure.TagReceiveOnReception)

	jww.INFO.Printf("[%v]: RID %d PostPhase FROM \"%s\" FOR \"%s\" RECIEVE/START", instance,
		roundID, phaseTy, p.GetType())
	//queue the phase to be operated on if it is not queued yet
	p.AttemptToQueue(instance.GetResourceQueue().GetPhaseQueue())

	//HACK HACK HACK
	//The share phase needs a batchsize of 1, when it receives
	// from generation on the first node this will do the
	// conversion on the batch
	if p.GetType() == phase.PrecompShare && len(batch.Slots) != 1 {
		batch.Slots = batch.Slots[:1]
		batch.Slots[0].PartialRoundPublicCypherKey =
			instance.GetConsensus().GetCmixGroup().GetG().Bytes()
		jww.INFO.Printf("[%v]: RID %d PostPhase PRECOMP SHARE HACK "+
			"HACK HACK", instance, roundID)
	}

	batch.FromPhase = int32(p.GetType())
	//send the data to the phase
	err = io.PostPhase(p, batch)

	if err != nil {
		jww.FATAL.Panicf("Error on PostPhase comm, should be"+
			" able to return: %+v", err)
	}
	return nil
}

// ReceiveStreamPostPhase handles the state checks and edge checks of
// receiving a phase operation
func ReceiveStreamPostPhase(streamServer mixmessages.Node_StreamPostPhaseServer,
	instance *server.Instance, auth *connect.Auth) error {

	// Get batch info
	batchInfo, err := node.GetPostPhaseStreamHeader(streamServer)
	if err != nil {
		return err
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
	if !auth.IsAuthenticated || prevNodeID.String() != auth.Sender.GetId() {
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
		jww.FATAL.Panicf("[%v]: Error on reception of "+
			"StreamPostPhase comm, should be able to return: \n %+v",
			instance, err)
	}
	p.Measure(measure.TagReceiveOnReception)

	jww.INFO.Printf("[%v]: RID %d StreamPostPhase FROM \"%s\" TO \"%s\" RECIEVE/START", instance,
		roundID, phaseTy, p.GetType())

	//queue the phase to be operated on if it is not queued yet
	p.AttemptToQueue(instance.GetResourceQueue().GetPhaseQueue())

	strmErr := io.StreamPostPhase(p, batchInfo.BatchSize, streamServer)

	return strmErr

}
