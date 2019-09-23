package io

import (
	"errors"
	"fmt"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server/round"
)

func TransmitRoundTripPing(network *node.NodeComms, id *id.Node, r *round.Round) error {
	roundID := r.GetID()

	r.StartRoundTrip()

	_, err := network.RoundTripPing(id, uint64(roundID))
	if err != nil{
		err = errors.New(fmt.Sprintf("TransmitRoundTripPing received an error: %+v", err))
		return err
	}

	return nil
}
