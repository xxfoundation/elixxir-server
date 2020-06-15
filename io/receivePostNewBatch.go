///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package io

// receivePostNewBatch.go contains the handler for the gateway <-> server postNewBatch comm

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

type GenericPostPhase func(p phase.Phase, batch *mixmessages.Batch) error

// Receive PostNewBatch comm from the gateway
// This should include an entire new batch that's ready for realtime processing
func ReceivePostNewBatch(instance *internal.Instance,
	newBatch *mixmessages.Batch, postPhase GenericPostPhase, auth *connect.Auth) error {

	// Check that authentication is good and the sender is our gateway, otherwise error
	if !auth.IsAuthenticated || !auth.Sender.GetId().Cmp(instance.GetGateway()) {
		jww.WARN.Printf("[%v]: ReceivePostNewBatch failed auth (sender ID: %s, auth: %v, expected: %s)",
			instance, auth.Sender.GetId(), auth.IsAuthenticated, instance.GetGateway().String())
		return connect.AuthError(auth.Sender.GetId())
	}

	// Wait for state to be REALTIME
	curActivity, err := instance.GetStateMachine().WaitFor(250*time.Millisecond, current.REALTIME)
	if err != nil {
		jww.WARN.Printf("Failed to transfer to realtime in time: %v", err)
		return errors.WithMessagef(err, errFailedToWait, current.REALTIME.String())
	}
	if curActivity != current.REALTIME {
		return errors.Errorf(errCouldNotWait, current.REALTIME.String())
	}

	nodeIDs, err := id.NewIDListFromBytes(newBatch.Round.Topology)
	if err != nil {
		return errors.Errorf("Unable to convert topology into a node list: %+v", err)
	}

	// fixme: this panics on error, external comm should not be able to crash server
	circuit := connect.NewCircuit(nodeIDs)

	if circuit.IsFirstNode(instance.GetID()) {
		err = HandleRealtimeBatch(instance, newBatch, postPhase)
		if err != nil {
			return err
		}
	}

	jww.INFO.Printf("[%v]: RID %d PostNewBatch END", instance,
		newBatch.Round.ID)

	return nil
}

// HandleRealtimeBatch is a helper function which handles phase and state operations
//  as well as calling postPhase for starting REALTIME
func HandleRealtimeBatch(instance *internal.Instance, newBatch *mixmessages.Batch, postPhase GenericPostPhase) error {
	// Get the roundinfo object
	ri := newBatch.Round
	rm := instance.GetRoundManager()
	rid := ri.GetRoundId()
	rnd, err := rm.GetRound(rid)
	if err != nil {
		return errors.WithMessage(err, "Failed to get round object from manager")
	}

	jww.INFO.Printf("[%v]: RID %d PostNewBatch START", instance,
		ri.ID)

	if uint32(len(newBatch.Slots)) != rnd.GetBuffer().GetBatchSize() {
		roundErr := errors.Errorf("[%v]: RID %d PostNewBatch ERROR - Gateway sent "+
			"batch with improper size", instance, newBatch.Round.ID)
		instance.ReportRoundFailure(roundErr, instance.GetID(), rid)
	}

	p, err := rnd.GetPhase(phase.RealDecrypt)
	if err != nil {
		roundErr := errors.Errorf("[%v]: RID %d Error on incoming PostNewBatch comm, could "+
			"not find phase \"%s\": %v", instance, newBatch.Round.ID,
			phase.RealDecrypt, err)
		instance.ReportRoundFailure(roundErr, instance.GetID(), rid)
	}

	if p.GetState() != phase.Active {
		roundErr := errors.Errorf("[%v]: RID %d Error on incoming PostNewBatch comm, phase "+
			"\"%s\" at incorrect state (\"%s\" vs \"Active\")", instance,
			newBatch.Round.ID, phase.RealDecrypt, p.GetState())
		instance.ReportRoundFailure(roundErr, instance.GetID(), rid)
	}

	p.Measure(measure.TagReceiveOnReception)

	// Queue the phase if it hasn't been done yet
	p.AttemptToQueue(instance.GetResourceQueue().GetPhaseQueue())

	err = postPhase(p, newBatch)

	if err != nil {
		roundErr := errors.Errorf("[%v]: RID %d Error on incoming PostNewBatch comm at"+
			" io PostPhase: %+v", instance, newBatch.Round.ID, err)
		instance.ReportRoundFailure(roundErr, instance.GetID(), rid)
	}

	return nil
}
