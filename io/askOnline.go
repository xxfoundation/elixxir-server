////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/comms/clusterclient"
	"time"
)

// Blocks until all given servers respond
func VerifyServersOnline(servers []string) {
	for i := 0; i < len(servers); {
		jww.DEBUG.Printf("Sending AskOnline message to %s...", servers[i])
		_, err := clusterclient.SendAskOnline(servers[i], &pb.Ping{})
		if err != nil {
			jww.ERROR.Printf("%v: Server %s failed to respond!", i, servers[i])
			time.Sleep(250 * time.Millisecond)
		} else {
			jww.DEBUG.Printf("%v: Server %s responded!", i, servers[i])
			i++
		}
	}
}
