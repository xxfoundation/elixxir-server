package node

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/io"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/server/phase"
)

func MakeStarter(batchSize uint32) server.RoundStarter {
	localBatchSize := batchSize
	return func(instance *server.Instance, rid id.Round) error {
		newBatch := &mixmessages.Batch{
			Slots:     make([]*mixmessages.Slot, localBatchSize),
			FromPhase: int32(phase.PrecompGeneration),
			Round: &mixmessages.RoundInfo{
				ID: uint64(rid),
			},
		}
		for i := 0; i < int(localBatchSize); i++ {
			newBatch.Slots[i] = &mixmessages.Slot{}
		}

		//get the round from the instance
		rm := instance.GetRoundManager()
		r, err := rm.GetRound(rid)

		if err != nil {
			jww.CRITICAL.Panicf("First Node Round Init: Could not get "+
				"round (%v) right after round init", rid)
		}

		// Do a round trip ping if we are the first node
		topology := r.GetTopology()
		myID := instance.GetID()
		if topology.IsFirstNode(myID) {
			nextNode := topology.GetNextNode(myID)
			err = io.TransmitRoundTripPing(instance.GetNetwork(), nextNode,
				r, &mixmessages.Ack{}, "EMPTY/ACK")
			if err != nil {
				jww.WARN.Printf("Failed to transmit round trip ping: %+v", err)
			}
		}

		//get the phase
		p := r.GetCurrentPhase()

		//queue the phase to be operated on if it is not queued yet
		p.AttemptToQueue(instance.GetResourceQueue().GetPhaseQueue())

		p.Measure(measure.TagReceiveOnReception)

		//send the data to the phase
		err = io.PostPhase(p, newBatch)

		if err != nil {
			jww.ERROR.Panicf("Error first node generation init: "+
				"should be able to return: %+v", err)
		}
		return nil
	}
}
