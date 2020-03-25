////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package receivers

import (
	"github.com/pkg/errors"
	"github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/server/server"
	"strings"
)

// Handles incoming Poll gateway responses, compares our NDF with the existing ndf
func ReceivePoll(poll *mixmessages.ServerPoll, instance *server.Instance) (*mixmessages.ServerPollResponse, error) {
	res := mixmessages.ServerPollResponse{}
	var err error
	// Node is only ready for a response once it has polled permissioning
	if instance.IsReadyForGateway() {
		network := instance.GetConsensus()
		//Compare partial NDF hash with instance and return the new one if they do not match
		isSame := network.GetPartialNdf().CompareHash(poll.GetPartial().Hash)
		if !isSame {
			res.PartialNDF = network.GetPartialNdf().GetPb()
		}

		//Compare Full NDF hash with instance and return the new one if they do not match
		isSame = network.GetFullNdf().CompareHash(poll.GetFull().Hash)
		if !isSame {
			res.FullNDF = network.GetFullNdf().GetPb()
		}

		// Populate the id field
		res.Id = instance.GetID().Bytes()

		//Check if any updates where made and get them
		res.Updates = network.GetRoundUpdates(int(poll.LastUpdate))

		// Get the request for a new batch que and store it into res
		if instance.GetStateMachine().Get() == current.REALTIME {
			res.BatchRequest, err = instance.GetRequestNewBatchQueue().Receive()
			if err != nil {
				jwalterweatherman.WARN.Printf("Failed to receive round info in realtime: %+v", err)
			}
		}
		res.Slots, err = GetCompletedBatch(instance)
		instance.GetGatewayFirstTime().Send()
		return &res, err
	}

	// If node has not gotten a response from permissioning, return an empty message
	return &res, errors.New("Node is not ready for gateway polling")
}

// GetCompletedBatch is used to return completed batches
func GetCompletedBatch(instance *server.Instance) ([]*mixmessages.Slot, error) {
	jwalterweatherman.DEBUG.Printf("Polling gateway for batch")
	// Check if a completed batch is ready to be returned, get the batch and return it if it is
	cr, err := instance.GetCompletedBatchQueue().Receive()
	if err != nil && !strings.Contains(err.Error(), "Did not recieve a completed round") {
		return nil, errors.Errorf("Unable to receive from CompletedBatchQueue: %+v", err)
	}
	var Slots []*mixmessages.Slot
	if cr != nil {

		r, err := instance.GetRoundManager().GetRound(cr.RoundID)
		if err != nil {
			return nil, errors.Errorf("Recieved completed batch for round %v that doesn't exist: %s", cr.RoundID, err)
		} else {
			Slots = make([]*mixmessages.Slot, r.GetBatchSize())
			// wait for everything from the channel then put it into a slot and return it
			for chunk := range cr.Receiver {
				for c := chunk.Begin(); c < chunk.End(); c++ {
					Slots[c] = cr.GetMessage(c)
				}
			}
		}
	}

	return Slots, nil

}
