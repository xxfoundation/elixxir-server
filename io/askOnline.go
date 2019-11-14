////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package io

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/connect"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"time"
)

// VerifyServersOnline Blocks until all given servers respond
func VerifyServersOnline(network *node.Comms, servers *connect.Circuit) {
	for i := 0; i < servers.Len(); {
		// Pull server's host from the connection manager
		serverID := servers.GetNodeAtIndex(i).String()
		server, ok := network.Manager.GetHost(serverID)
		if !ok {
			jww.INFO.Printf("Could not find cMix server %s (%d/%d) in comm manager", serverID, i+1, servers.Len())
		}
		// Send comm to the other server
		_, err := network.SendAskOnline(server, &pb.Ping{})
		jww.INFO.Printf("Waiting for cMix server %s (%d/%d)...",
			serverID, i+1, servers.Len())
		if err != nil {
			jww.INFO.Printf("Could not contact cMix server %s (%d/%d)...",
				serverID, i+1, servers.Len())
			time.Sleep(250 * time.Millisecond)
		} else {
			i++
		}
	}
}
