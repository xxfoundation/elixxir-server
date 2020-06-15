///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package io

// ReceiveRoundTripPing.go contains the handler for RoundTripPing

import (
	"github.com/pkg/errors"
	"github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/internal"
)

// ReceiveRoundTripPing handles incoming round trip pings, stopping the ping when back at the first node
func ReceiveRoundTripPing(instance *internal.Instance, msg *mixmessages.RoundTripPing) error {

	// Copy out the topology to prevent any data races
	topologyBytes := make([][]byte, len(msg.Round.Topology))
	for i := 0; i < len(topologyBytes); i++ {
		topologyBytes[i] = make([]byte, len(msg.Round.Topology[i]))
		copy(topologyBytes[i], msg.Round.Topology[i])
	}
	nodeIDs, err := id.NewIDListFromBytes(topologyBytes)
	if err != nil {
		return errors.Errorf("Unable to convert topology into a node list: %+v", err)
	}

	topology := connect.NewCircuit(nodeIDs)
	myID := instance.GetID()

	roundID := msg.Round.ID

	if topology.IsFirstNode(myID) {
		r, err := instance.GetRoundManager().GetRound(id.Round(roundID))
		if err != nil {
			err = errors.Errorf("ReceiveRoundTripPing could not get round: %+v", err)
			return err
		}

		err = r.StopRoundTrip()
		if err != nil {
			err = errors.Errorf("ReceiveRoundTrip failed to stop round trip: %+v", err)
			jwalterweatherman.ERROR.Println(err.Error())
			return err
		}
		return nil
	}

	// Pull the particular server host object from the commManager
	nextNodeID := topology.GetNextNode(instance.GetID())
	nextNode, ok := instance.GetNetwork().GetHost(nextNodeID)
	if !ok {
		jwalterweatherman.ERROR.Printf("Could not find next node [%v]:", nextNode)
		return errors.Errorf("Could not find next node [%v]:", nextNode)
	}

	//Send the round trip ping to the next node
	_, err = instance.GetNetwork().RoundTripPing(nextNode, msg)
	if err != nil {
		err = errors.Errorf("ReceiveRoundTripPing failed to send ping to next node: %+v", err)
		return err
	}

	return nil
}
