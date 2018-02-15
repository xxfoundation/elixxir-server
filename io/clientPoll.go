package io

import (
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/server/globals"
)

// Determine whether a message is in the buffer for a given User
// Return the message if so or a blank message if not
func (s ServerImpl) ClientPoll(inputMsg *pb.ClientPollMessage) *pb.CmixMessage {
	user := globals.Users.GetUser(inputMsg.UserID)
	select {
	case msg := <-user.MessageBuffer:
		jww.DEBUG.Println("Message pending for User %s", user.Id)
		return msg
	default:
		jww.DEBUG.Printf("No messages pending for User %s!", user.Id)
		return &pb.CmixMessage{}
	}
}
