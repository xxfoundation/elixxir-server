///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package io

// receiveUploadUnmixedBatch.go contains the handler for the gateway <-> server postNewBatch comm

import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/primitives/id"
	"io"
	"time"
)

type GenericPostPhase func(p phase.Phase, batch *mixmessages.Batch) error

// ReceiveUploadUnmixedBatchStream handler from the gateway
// This should include an entire new batch that's ready for realtime processing
func ReceiveUploadUnmixedBatchStream(instance *internal.Instance,
	stream mixmessages.Node_UploadUnmixedBatchServer, postPhase GenericPostPhase,
	auth *connect.Auth) error {

	// Check that authentication is good and the sender is our gateway, otherwise error
	if !auth.IsAuthenticated || !auth.Sender.GetId().Cmp(instance.GetGateway()) {
		jww.WARN.Printf("[%v]: ReceiveUploadUnmixedBatchStream failed auth (sender ID: %s, auth: %v, expected: %s)",
			instance, auth.Sender.GetId(), auth.IsAuthenticated, instance.GetGateway().String())
		return connect.AuthError(auth.Sender.GetId())
	}

	// Extract header information
	batchInfo, err := node.GetUnmixedBatchStreamHeader(stream)
	if err != nil {
		return errors.WithMessage(err, "Could not get unmixed batch stream header")
	}

	// Receive the stream
	_, newBatch, strmErr := receiveUploadUnmixedBatch(stream, batchInfo)
	if strmErr != nil {
		jww.ERROR.Printf("SteamUnmixedBatch error: %v", strmErr)
		return strmErr
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

	jww.INFO.Printf("[%v]: RID %d receiveUploadUnmixedBatch END", instance,
		newBatch.Round.ID)
	// fixme: this is not for node -> node like streamPostPhase, but may be useful?
	//phaseTy := phase.Type(batchInfo.FromPhase).String()
	//roundID := id.Round(batchInfo.Round.ID)
	//jww.INFO.Printf("\tbwLogging: Round %d, "+
	//	"received phase: %s, "+
	//	"from: %s, to: %s, "+
	//	"started: %v, "+
	//	"ended: %v, "+
	//	"duration: %d,",
	//	roundID, phaseTy,
	//	auth.Sender.GetId().String(), instance.GetID(),
	//	streamInfo.Start, streamInfo.End,
	//	streamInfo.End.Sub(streamInfo.Start).Milliseconds())

	return nil
}

// receiveUploadUnmixedBatch is a helper function which receives the
// streaming slots and builds a batch.
func receiveUploadUnmixedBatch(stream mixmessages.Node_UploadUnmixedBatchServer,
	batchInfo *mixmessages.BatchInfo) (*streamInfo, *mixmessages.Batch, error) {

	newBatch := &mixmessages.Batch{
		Round:     batchInfo.Round,
		FromPhase: batchInfo.FromPhase,
	}

	batchSize := batchInfo.BatchSize

	// Receive the slots
	slot, err := stream.Recv()
	var start, end time.Time
	slots := make([]*mixmessages.Slot, 0)
	slotsReceived := uint32(0)
	for ; err == nil; slot, err = stream.Recv() {
		slotsReceived++
		if slotsReceived == 1 {
			start = time.Now()
		}
		if slotsReceived == batchSize {
			end = time.Now()
		}
		slots = append(slots, slot)
	}

	newBatch.Slots = slots

	// Handle any errors
	ack := &messages.Ack{Error: ""}
	if err != io.EOF {
		ack.Error = fmt.Sprintf("errors occurred, %v/%v slots "+
			"recived: %s", slotsReceived, batchSize, err.Error())
	} else if slotsReceived != batchSize {
		ack.Error = fmt.Sprintf("Mismatch between batch size %v "+
			"and received num slots %v, no error", slotsReceived, batchSize)
	}

	// Close the stream by sending ack and returning success or failure
	si := &streamInfo{Start: start, End: end}
	errClose := stream.SendAndClose(ack)
	if errClose != nil && ack.Error != "" {
		return si, newBatch, errors.WithMessage(errClose, ack.Error)
	} else if errClose == nil && ack.Error != "" {
		return si, newBatch, errors.New(ack.Error)
	} else {
		return si, newBatch, errClose
	}

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
		instance.ReportRoundFailure(roundErr, instance.GetID(), rid, false)
	}

	p, err := rnd.GetPhase(phase.RealDecrypt)
	if err != nil {
		roundErr := errors.Errorf("[%v]: RID %d Error on incoming PostNewBatch comm, could "+
			"not find phase \"%s\": %v", instance, newBatch.Round.ID,
			phase.RealDecrypt, err)
		instance.ReportRoundFailure(roundErr, instance.GetID(), rid, false)
	}

	if p.GetState() != phase.Active {
		roundErr := errors.Errorf("[%v]: RID %d Error on incoming PostNewBatch comm, phase "+
			"\"%s\" at incorrect state (\"%s\" vs \"Active\")", instance,
			newBatch.Round.ID, phase.RealDecrypt, p.GetState())
		instance.ReportRoundFailure(roundErr, instance.GetID(), rid, false)
	}

	p.Measure(measure.TagReceiveOnReception)

	// Queue the phase if it hasn't been done yet
	p.AttemptToQueue(instance.GetResourceQueue().GetPhaseQueue())

	err = postPhase(p, newBatch)

	if err != nil {
		roundErr := errors.Errorf("[%v]: RID %d Error on incoming PostNewBatch comm at"+
			" io PostPhase: %+v", instance, newBatch.Round.ID, err)
		instance.ReportRoundFailure(roundErr, instance.GetID(), rid, false)
	}

	return nil
}
