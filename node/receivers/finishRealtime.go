package receivers

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/server/phase"
	"time"
)

// ReceiveFinishRealtime handles the state checks and edge checks of
// receiving the signal that the realtime has completed
func ReceiveFinishRealtime(instance *server.Instance, msg *mixmessages.RoundInfo,
	auth *connect.Auth) error {
	ok, err := instance.GetStateMachine().WaitFor(current.REALTIME, 250)
	if err != nil {
		return errors.WithMessagef(err, errFailedToWait, current.REALTIME.String())
	}
	if !ok {
		return errors.Errorf(errCouldNotWait, current.REALTIME.String())
	}

	//check that the round should have finished and return it
	roundID := id.Round(msg.ID)
	rm := instance.GetRoundManager()
	r, err := rm.GetRound(roundID)
	if err != nil {
		return errors.WithMessage(err, "Failed to get round")
	}

	expectedID := r.GetTopology().GetLastNode()
	if !auth.IsAuthenticated || auth.Sender.GetId() != expectedID.String() {
		jww.INFO.Printf("[%s]: RID %d FinishRealtime failed auth "+
			"(expected ID: %s, received ID: %s, auth: %v)",
			instance, roundID, expectedID, auth.Sender.GetId(),
			auth.IsAuthenticated)
		return connect.AuthError(auth.Sender.GetId())
	}

	jww.INFO.Printf("[%s]: RID %d ReceiveFinishRealtime START",
		instance, roundID)

	tag := phase.RealPermute.String() + "Verification"
	r, p, err := rm.HandleIncomingComm(id.Round(roundID), tag)
	if err != nil {
		jww.FATAL.Panicf("[%s]: Error on reception of "+
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
				jww.INFO.Printf("[%s]: RID %d ReceiveFinishRealtime CLEARING "+
					"CMIX BUFFERS", instance, roundID)

				time.Sleep(time.Duration(60) * time.Second)
				r.GetBuffer().Erase()
				rm.DeleteRound(roundID)
			}()

		} else {
			jww.WARN.Printf("[%s]: RID %d ReceiveFinishRealtime MEMORY "+
				"LEAK - Round buffers not purged ", instance,
				roundID)
		}
	}()

	jww.INFO.Printf("[%s]: RID %d ReceiveFinishRealtime END", instance,
		roundID)

	jww.INFO.Printf("[%s]: RID %d Round took %v seconds",
		instance, roundID, time.Now().Sub(r.GetTimeStart()))

	//Send batch to Gateway Polling Receiver on last node
	if r.GetTopology().IsLastNode(instance.GetID()) {

	}

	//Send the finished signal on first node
	if r.GetTopology().IsFirstNode(instance.GetID()) {
		jww.INFO.Printf("[%s]: RID %d FIRST NODE ReceiveFinishRealtime"+
			" SENDING END ROUND SIGNAL", instance, roundID)

		//instance.FinishRound(roundID)

	}
	select {
	case r.GetMeasurementsReadyChan() <- struct{}{}:
	default:
	}

	return nil
}
