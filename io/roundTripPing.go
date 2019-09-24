package io

import (
	"errors"
	"fmt"
	"github.com/golang/protobuf/ptypes"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server/round"
)

// TransmitRoundTripPing sends a round trip ping and starts
func TransmitRoundTripPing(network *node.NodeComms, id *id.Node, r *round.Round) error {
	roundID := r.GetID()

	r.StartRoundTrip()

	any, err := ptypes.MarshalAny(&mixmessages.Ack{})
	if err != nil {
		err = errors.New(fmt.Sprintf("TransmitRoundTripPing: failed attempting to marshall any type: %+v", err))
	}

	_, err = network.RoundTripPing(id, uint64(roundID), any)
	if err != nil {
		err = errors.New(fmt.Sprintf("TransmitRoundTripPing received an error: %+v", err))
		return err
	}

	return nil
}
