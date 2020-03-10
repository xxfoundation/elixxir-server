////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package receivers

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/server/server"
)

// Handles incoming Poll gateway responses, compares our NDF with the existing ndf
func ReceivePoll(poll *mixmessages.ServerPoll, instance *server.Instance) (*mixmessages.ServerPollResponse, error) {

	res := mixmessages.ServerPollResponse{}

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

	//Check if any updates where made and get them
	res.Updates = network.GetRoundUpdates(int(poll.LastUpdate))

	// Get the request for a new batch que and store it into res
	if instance.GetStateMachine().Get() == current.REALTIME{
		res.BatchRequest, _ = instance.GetRequestNewBatchQueue().Receive()
	}

	// Get a Batch message and store it into res
	cr := instance.GetCompletedBatchQueue().Recieve()
	if cr != nil {
		r, err := instance.GetRoundManager().GetRound(cr.RoundID)
		if err != nil {
			jww.ERROR.Printf("Recieved completed batch for round %v that doesn't exist: %s", cr.RoundID, err)
		} else {
			res.Slots = make([]*mixmessages.Slot, r.GetBatchSize())
			// wait for everything from the channel then put it into a slot and return it
			for chunk := range cr.Receiver {
				for c := chunk.Begin(); c < chunk.End(); c++ {
					res.Slots[c] = cr.GetMessage(c)
				}
			}
		}
	}

	return &res, nil
}
