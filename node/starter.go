package node

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/io"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/server/phase"
	insecureRand "math/rand"
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

			payloadInfo := "EMPTY/ACK"
			var payload proto.Message
			payload = &mixmessages.Ack{}
			//does not work properly because it doesnt use streaming comms
			/*
				if rid%FullTestFrequency == 0 {
					p, err := buildBatchRTPingPayload(batchSize)

					if err != nil {
						jww.WARN.Printf("Could not build batch payload to "+
							"transmit on 50th round for round trip ping, "+
							"transmitting blank instead: %+v", err)
					}

					payload = p
					payloadInfo = "FULL/BATCH"
				}*/

			nextNode := topology.GetNextNode(myID)
			err = io.TransmitRoundTripPing(instance.GetNetwork(), nextNode,
				r, payload, payloadInfo)
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

//buildBatchRTPingPayload builds a fake batch to use for testing of full
//communications. unused for now
func buildBatchRTPingPayload(batchsize uint32) (proto.Message, error) {

	payload := &mixmessages.Batch{}
	payload.Slots = make([]*mixmessages.Slot, batchsize)

	for i := uint32(0); i < batchsize; i++ {
		A := make([]byte, 256)
		B := make([]byte, 256)
		salt := make([]byte, 32)
		sid := make([]byte, 32)

		_, err := insecureRand.Read(A)
		if err != nil {
			err = errors.New(fmt.Sprintf("TransmitRoundTripPing: failed to generate random bytes A: %+v", err))
			return nil, err
		}
		_, err = insecureRand.Read(B)
		if err != nil {
			err = errors.New(fmt.Sprintf("TransmitRoundTripPing: failed to generate random bytes B: %+v", err))
			return nil, err
		}
		_, err = insecureRand.Read(salt)
		if err != nil {
			err = errors.New(fmt.Sprintf("TransmitRoundTripPing: failed to generate random bytes B: %+v", err))
			return nil, err
		}
		_, err = insecureRand.Read(sid)
		if err != nil {
			err = errors.New(fmt.Sprintf("TransmitRoundTripPing: failed to generate random bytes for id: %+v", err))
			return nil, err
		}

		slot := mixmessages.Slot{
			SenderID: sid,
			PayloadA: A,
			PayloadB: B,
			Salt:     salt,
		}
		payload.Slots[i] = &slot
	}

	return payload, nil
}
