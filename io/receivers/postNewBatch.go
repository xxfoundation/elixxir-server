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
	"gitlab.com/elixxir/server/internals"
	"gitlab.com/elixxir/server/internals/measure"
	"gitlab.com/elixxir/server/internals/phase"
	"time"
)

type PostPhase func(p phase.Phase, batch *mixmessages.Batch) error

// Receive PostNewBatch comm from the gateway
// This should include an entire new batch that's ready for realtime processing
func ReceivePostNewBatch(instance *server.Instance,
	newBatch *mixmessages.Batch, postPhase PostPhase, auth *connect.Auth) error {

	// Check that authentication is good and the sender is our gateway, otherwise error
	if !auth.IsAuthenticated || auth.Sender.GetId() != instance.GetGateway().String() {
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

	nodeIDs, err := id.NewNodeListFromStrings(newBatch.Round.Topology)
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
func HandleRealtimeBatch(instance *server.Instance, newBatch *mixmessages.Batch, postPhase PostPhase) error {
	// Get the roundinfo object
	ri := newBatch.Round
	rm := instance.GetRoundManager()
	rnd, err := rm.GetRound(ri.GetRoundId())
	if err != nil {
		return errors.WithMessage(err, "Failed to get round object from manager")
	}

	jww.INFO.Printf("[%v]: RID %d PostNewBatch START", instance,
		ri.ID)

	if uint32(len(newBatch.Slots)) != rnd.GetBuffer().GetBatchSize() {
		jww.FATAL.Panicf("[%v]: RID %d PostNewBatch ERROR - Gateway sent "+
			"batch with improper size", instance, ri.GetID())
	}

	p, err := rnd.GetPhase(phase.RealDecrypt)

	if err != nil {
		jww.FATAL.Panicf(
			"[%v]: RID %d Error on incoming PostNewBatch comm, could "+
				"not find phase \"%s\": %v", instance, newBatch.Round.ID,
			phase.RealDecrypt, err)
	}

	if p.GetState() != phase.Active {
		jww.FATAL.Panicf(
			"[%v]: RID %d Error on incoming PostNewBatch comm, phase "+
				"\"%s\" at incorrect state (\"%s\" vs \"Active\")", instance,
			newBatch.Round.ID, phase.RealDecrypt, p.GetState())
	}

	p.Measure(measure.TagReceiveOnReception)

	// Queue the phase if it hasn't been done yet
	p.AttemptToQueue(instance.GetResourceQueue().GetPhaseQueue())

	err = postPhase(p, newBatch)

	if err != nil {
		jww.FATAL.Panicf("[%v]: RID %d Error on incoming PostNewBatch comm at"+
			" io PostPhase: %+v", instance, newBatch.Round.ID, err)
	}

	return nil
}
