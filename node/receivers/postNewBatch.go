package receivers

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/server/io"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/server/phase"
)

// Receive PostNewBatch comm from the gateway
// This should include an entire new batch that's ready for realtime processing
func ReceivePostNewBatch(instance *server.Instance,
	newBatch *mixmessages.Batch, auth *connect.Auth) error {
	// Check that authentication is good and the sender is our gateway, otherwise error
	if !auth.IsAuthenticated || auth.Sender.GetId() != instance.GetGateway().String() {
		jww.WARN.Printf("[%v]: ReceivePostNewBatch failed auth (sender ID: %s, auth: %v, expected: %s)",
			instance, auth.Sender.GetId(), auth.IsAuthenticated, instance.GetGateway().String())
		return connect.AuthError(auth.Sender.GetId())
	}

	// Wait for state to be REALTIME
	ok, err := instance.GetStateMachine().WaitFor(current.REALTIME, 250)
	if err != nil {
		return errors.WithMessagef(err, errFailedToWait, current.REALTIME.String())
	}
	if !ok {
		return errors.Errorf(errCouldNotWait, current.REALTIME.String())
	}

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
			"batch with improper size", instance, newBatch.Round.ID)
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
	for i := range newBatch.Slots {
		jww.DEBUG.Printf("new Batch: %#v", newBatch.Slots[i])
	}
	err = io.PostPhase(p, newBatch)

	if err != nil {
		jww.FATAL.Panicf("[%v]: RID %d Error on incoming PostNewBatch comm at"+
			" io PostPhase: %+v", instance, newBatch.Round.ID, err)
	}

	jww.INFO.Printf("[%v]: RID %d PostNewBatch END", instance,
		newBatch.Round.ID)

	return nil
}
