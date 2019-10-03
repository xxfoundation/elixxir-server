package io

import (
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server/round"
)

// TransmitRoundTripPing sends a round trip ping and starts
func TransmitRoundTripPing(network *node.NodeComms, id *id.Node, r *round.Round, fullBatch bool) error {
	roundID := r.GetID()

	var anyPayload proto.Message
	var payloadInfo string
	if fullBatch {
		A := make([]byte, 256)
		B := make([]byte, 256)
		salt := make([]byte, 32)
		_, err := rand.Read(A)
		if err != nil {
			err = errors.New(fmt.Sprintf("TransmitRoundTripPing: failed to generate random bytes A: %+v", err))
			return err
		}
		_, err = rand.Read(B)
		if err != nil {
			err = errors.New(fmt.Sprintf("TransmitRoundTripPing: failed to generate random bytes B: %+v", err))
			return err
		}
		_ , err = rand.Read(salt)
		if err != nil {
			err = errors.New(fmt.Sprintf("TransmitRoundTripPing: failed to generate random bytes B: %+v", err))
			return err
		}
		anyPayload = &mixmessages.Batch{
			Slots: []*mixmessages.Slot{
				{
					SenderID: id.Bytes(),
					PayloadA: A,
					PayloadB: B,
					// Because the salt is just one byte,
					// this should fail in the Realtime Decrypt graph.
					Salt:  salt,
				},
			},
		}
		payloadInfo = "FULL/BATCH"
	} else {
		anyPayload = &mixmessages.Ack{}
		payloadInfo = "EMPTY/ACK"
	}

	jwalterweatherman.DEBUG.Printf("Sending round trip ping with payload %s", payloadInfo)

	any, err := ptypes.MarshalAny(anyPayload)
	if err != nil {
		err = errors.New(fmt.Sprintf("TransmitRoundTripPing: failed attempting to marshall any type: %+v", err))
		return err
	}
	r.StartRoundTrip(payloadInfo)

	_, err = network.RoundTripPing(id, uint64(roundID), any)
	if err != nil {
		err = errors.New(fmt.Sprintf("TransmitRoundTripPing received an error: %+v", err))
		return err
	}

	return nil
}
