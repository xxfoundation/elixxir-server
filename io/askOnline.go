////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package io

import (
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/circuit"
	"time"
)

// VerifyServersOnline Blocks until all given servers respond
func VerifyServersOnline(comms *node.NodeComms, servers *circuit.Circuit) {
	for i := 0; i < servers.Len(); {
		server := servers.GetNodeAtIndex(i)
		_, err := comms.SendAskOnline(server, &pb.Ping{})
		jww.INFO.Printf("Waiting for cMix server %s (%d/%d)...",
			server, i+1, servers.Len())
		if err != nil {
			jww.INFO.Printf("Could not contact cMix server %s (%d/%d)...",
				server, i+1, servers.Len())
			time.Sleep(250 * time.Millisecond)
		} else {
			i++
		}
	}
}
