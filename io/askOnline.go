////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"time"
)

// Blocks until all given servers respond
func VerifyServersOnline() {
	for i := 0; i < len(Servers); {
		_, err := node.SendAskOnline(Servers[i], &pb.Ping{})
		jww.INFO.Printf("Waiting for federation server %s (%d/%d)...",
			Servers[i], i+1, len(Servers))
		if err != nil {
			time.Sleep(250 * time.Millisecond)
		} else {
			i++
		}
	}
}
