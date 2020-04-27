package io

import (
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/internal/round"
)

// TransmitRoundTripPing sends a round trip ping and starts
func TransmitRoundTripPing(network *node.Comms, id *id.Node, r *round.Round,
	payload proto.Message, payloadInfo string, ri *mixmessages.RoundInfo) error {
	any, err := ptypes.MarshalAny(payload)
	if err != nil {
		err = errors.Errorf("TransmitRoundTripPing: failed attempting to marshall any type: %+v", err)
		return err
	}

	r.StartRoundTrip(payloadInfo)
	// Pull the particular server host object from the commManager
	recipient, ok := network.Manager.GetHost(id.String())
	if !ok {
		errMsg := errors.Errorf("Could not find cMix server %s in comm manager", id)
		return errMsg
	}

	rtPing := &mixmessages.RoundTripPing{
		Round:   ri,
		Payload: any,
	}

	// Send the round trip ping
	_, err = network.RoundTripPing(recipient, rtPing)
	if err != nil {
		err = errors.Errorf("TransmitRoundTripPing received an error: %+v", err)
		return err
	}
	return nil
}
