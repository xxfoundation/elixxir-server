package io

import (
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server/round"
)

// TransmitRoundTripPing sends a round trip ping and starts
func TransmitRoundTripPing(network *node.NodeComms, id *id.Node, r *round.Round, payload proto.Message, payloadInfo string) error {
	any, err := ptypes.MarshalAny(payload)
	if err != nil {
		err = errors.New(fmt.Sprintf("TransmitRoundTripPing: failed attempting to marshall any type: %+v", err))
		return err
	}
	r.StartRoundTrip(payloadInfo)

	_, err = network.RoundTripPing(id, uint64(r.GetID()), any)
	if err != nil {
		err = errors.New(fmt.Sprintf("TransmitRoundTripPing received an error: %+v", err))
		return err
	}

	return nil
}
