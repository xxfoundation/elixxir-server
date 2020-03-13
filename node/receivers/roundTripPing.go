package receivers

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server"
	"time"
)

// ReceiveRoundTripPing handles incoming round trip pings, stopping the ping when back at the first node
func ReceiveRoundTripPing(instance *server.Instance, msg *mixmessages.RoundTripPing) error {
	// Ensure that round trip ping is in the correct state
	ok, err := instance.GetStateMachine().WaitFor(current.PRECOMPUTING, 1*time.Second)
	if !ok || err != nil {
		jww.FATAL.Panicf("ReceiveRoundTripPing timed out in state transition to %v: %+v", current.PRECOMPUTING, err)
	}

	roundID := msg.Round.ID
	rm := instance.GetRoundManager()
	r, err := rm.GetRound(id.Round(roundID))
	if err != nil {
		err = errors.Errorf("ReceiveRoundTripPing could not get round: %+v", err)
		return err
	}

	//jww.INFO.Printf("Recieved RoundTripPing, payload size: %v", len(msg.Payload.Value))

	topology := r.GetTopology()
	myID := instance.GetID()

	if topology.IsFirstNode(myID) {
		err = r.StopRoundTrip()
		if err != nil {
			err = errors.Errorf("ReceiveRoundTrip failed to stop round trip: %+v", err)
			jww.ERROR.Println(err.Error())
			return err
		}
		return nil
	}

	// Pull the particular server host object from the commManager
	nextNodeID := topology.GetNextNode(myID)
	nextNodeIndex := topology.GetNodeLocation(nextNodeID)
	nextNode := topology.GetHostAtIndex(nextNodeIndex)

	//Send the round trip ping to the next node
	_, err = instance.GetNetwork().RoundTripPing(nextNode, roundID, msg.Payload)
	if err != nil {
		err = errors.Errorf("ReceiveRoundTripPing failed to send ping to next node: %+v", err)
		return err
	}

	return nil
}
