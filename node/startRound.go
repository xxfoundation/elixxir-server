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
	"gitlab.com/elixxir/server/server/round"
	insecureRand "math/rand"
)

func StartLocalPrecomp(instance *server.Instance, rid id.Round) error {
	//get the round from the instance
	rm := instance.GetRoundManager()

	r, err := rm.GetRound(rid)
	if err != nil {
		roundErr := errors.Errorf("First Node Round Init: Could not get "+
			"round (%v) right after round init", rid)
		instance.ReportRoundFailure(roundErr, instance.GetID(), &rid)
	}

	// Create new batch object
	batchSize := r.GetBatchSize()
	newBatch := &mixmessages.Batch{
		Slots:     make([]*mixmessages.Slot, batchSize),
		FromPhase: int32(phase.PrecompGeneration),
		Round: &mixmessages.RoundInfo{
			ID: uint64(rid),
		},
	}
	for i := 0; i < int(batchSize); i++ {
		newBatch.Slots[i] = &mixmessages.Slot{}
	}

	ourRoundInfo, err := instance.GetConsensus().GetRound(rid)
	if err != nil {
		roundErr := errors.Errorf("Could not get round info from instance: %v", err)
		instance.ReportRoundFailure(roundErr, instance.GetID(), &rid)
	}
	// Make this a non anonymous functions, that calls a new thread and test the function seperately
	go func() {
		_ = doRoundTripPing(r, instance, ourRoundInfo)
	}()

	//get the phase
	p := r.GetCurrentPhase()

	//queue the phase to be operated on if it is not queued yet
	p.AttemptToQueue(instance.GetResourceQueue().GetPhaseQueue())
	p.Measure(measure.TagReceiveOnReception)

	//send the data to the phase
	err = io.PostPhase(p, newBatch)
	if err != nil {
		roundErr := errors.Errorf("Error on processing new batch in phase %s of round %v: %s", p.GetType(), rid, err)
		return roundErr
	}
	return nil
}

func doRoundTripPing(round *round.Round, instance *server.Instance, ri *mixmessages.RoundInfo) error {
	payloadInfo := "EMPTY/ACK"
	var payload proto.Message
	payload = &mixmessages.Ack{}

	// Get create topology and fetch the next node
	topology := round.GetTopology()
	myID := instance.GetID()
	nextNode := topology.GetNextNode(myID)

	// Send rount trip ping to the next node
	err := io.TransmitRoundTripPing(instance.GetNetwork(), nextNode,
		round, payload, payloadInfo, ri)
	if err != nil {
		jww.WARN.Printf("Failed to transmit round trip ping: %+v", err)
		return err
	}

	return nil
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
