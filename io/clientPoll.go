package io

import (
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/server/globals"
)

// Determine whether a message is in the buffer for a given User
// Return the message if so or a blank message if not
func (s ServerImpl) ClientPoll(inputMsg *pb.ClientPollMessage) *pb.CmixMessage {
	user, userExists := globals.Users.GetUser(inputMsg.UserID)
	// Verify the User exists
	if userExists {
		select {
		case msg := <-user.MessageBuffer:
			// Return pending message for the given User
			jww.DEBUG.Println("Message pending for User %s", user.Id)
			return msg
		default:
			jww.DEBUG.Printf("No messages pending for User %s!", user.Id)
		}
	}
	// Return blank message if nonexistent User or no messages
	return &pb.CmixMessage{}
}
