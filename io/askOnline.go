////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/privategrity/comms/clusterclient"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"time"
)

// Blocks until all given servers respond
func VerifyServersOnline(servers []string) {
	for i := 0; i < len(servers); {
		_, err := clusterclient.SendAskOnline(servers[i], &pb.Ping{})
		jww.INFO.Printf("Waiting for other federation servers (%d/%d)...",
			i+1, len(servers))
		if err != nil {
			time.Sleep(250 * time.Millisecond)
		} else {
			i++
		}
	}
}
