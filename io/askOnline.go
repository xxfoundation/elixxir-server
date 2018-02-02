package io

import (
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/comms/mixserver/message"
)

// Checks to see if the given servers are online TODO make blocking
func verifyServersOnline(servers []string) {
	for i := range servers {
		_, err := message.SendAskOnline(servers[i], &pb.Ping{})
		if err != nil {
			jww.ERROR.Println("Server %s failed to respond!", servers[i])
		}
	}
}
