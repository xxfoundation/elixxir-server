///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package io

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/xx_network/primitives/id"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// TransmitRoundTripPing sends a round trip ping and starts
func TransmitRoundTripPing(network *node.Comms, id *id.ID, r *round.Round,
	payload proto.Message, payloadInfo string, ri *mixmessages.RoundInfo) error {
	any, err := anypb.New(payload)
	if err != nil {
		err = errors.Errorf("TransmitRoundTripPing: failed attempting to marshall any type: %+v", err)
		return err
	}

	r.StartRoundTrip(payloadInfo)
	// Pull the particular server host object from the commManager
	recipient, ok := network.Manager.GetHost(id)
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
