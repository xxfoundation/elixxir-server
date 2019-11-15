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
func TransmitRoundTripPing(network *node.Comms, id *id.Node, r *round.Round, payload proto.Message, payloadInfo string) error {
	any, err := ptypes.MarshalAny(payload)
	if err != nil {
		err = errors.New(fmt.Sprintf("TransmitRoundTripPing: failed attempting to marshall any type: %+v", err))
		return err
	}
	r.StartRoundTrip(payloadInfo)
	// Pull the particular server host object from the commManager
	recipient, ok := network.Manager.GetHost(id.String())
	if !ok {
		errMsg := fmt.Sprintf("Could not find cMix server %s in comm manager", id)
		return errors.New(errMsg)
	}
	// Send the round trip ping
	_, err = network.RoundTripPing(recipient, uint64(r.GetID()), any)
	if err != nil {
		err = errors.New(fmt.Sprintf("TransmitRoundTripPing received an error: %+v", err))
		return err
	}
	return nil
}
