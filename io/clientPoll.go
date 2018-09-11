////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/crypto/id"
)

// Determine whether a message is in the buffer for a given User
// Return the message if so or a blank message if not
func (s ServerImpl) ClientPoll(inputMsg *pb.ClientPollMessage) *pb.CmixMessage {
	var userId id.UserID
	copy(userId[:], inputMsg.UserID)
	user, err := globals.Users.GetUser(userId)
	// Verify the User exists
	if err == nil {
		select {
		case msg := <-user.MessageBuffer:
			// Return pending message for the given User
			jww.DEBUG.Printf("Message pending for User %v", user.ID)
			return msg
		default:
			jww.DEBUG.Printf("No messages pending for User %v!", user.ID)
		}
	}
	// Return blank message if nonexistent User or no messages
	return &pb.CmixMessage{}
}
