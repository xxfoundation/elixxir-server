////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package receivers

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server"
)

// ReceiveRoundError takes the round error message and checks if it's within the round
// If so then we transition to an error state. If not we ignore the error and send it
func ReceiveRoundError(msg *mixmessages.RoundError, auth *connect.Auth, instance *server.Instance) error {

	// Get the round information from message. If the sent round information is
	// invalid, then we send back an error
	roundId := msg.GetId()
	rm := instance.GetRoundManager()
	r, err := rm.GetRound(id.Round(roundId))
	if err != nil {
		return errors.WithMessagef(err, "Failed to get round %d", roundId)
	}

	// Pull the erroring node id from message and check if it's valid
	badNodeId, err := id.NewNodeFromString(msg.GetNodeId())
	if err != nil {
		return errors.Errorf("Received unrecognizable node id: %v",
			err.Error())
	}

	// Build topology from round information
	topology := r.GetTopology()

	// Attempt to pull the node from the topology
	nodeIndex := topology.GetNodeLocation(badNodeId)

	// Check for proper authentication and if the sender is in the circuit
	// a -1 value indicates a non existant node in the topology
	if !auth.IsAuthenticated || nodeIndex == -1 {
		jww.INFO.Printf("[%v]: RID %d RoundError failed auth "+
			"(expected ID: %s, received ID: %s, auth: %v)",
			instance, roundId, badNodeId, auth.Sender.GetId(),
			auth.IsAuthenticated)
		return connect.AuthError(auth.Sender.GetId())
	}

	jww.ERROR.Printf("ReceiveRoundError received error from [%v]: %+v. Transitioning to ERROR...",
		badNodeId, msg.Error)

	roundError := errors.New(msg.Error)
	rid := id.Round(roundId)
	instance.ReportRoundFailure(roundError, badNodeId, &rid)

	return nil
}
