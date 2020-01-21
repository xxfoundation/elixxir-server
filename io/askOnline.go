////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package io

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/node"
	"time"
)

// VerifyServersOnline Blocks until all given servers respond
func VerifyServersOnline(network *node.Comms, servers *connect.Circuit) {
	for i := servers.Len()-1; i >= 0;{
		// Pull server's host from the connection manager
		serverID := servers.GetNodeAtIndex(i)
		server := servers.GetHostAtIndex(i)

		// Send AskOnline to all servers
		jww.INFO.Printf("Waiting for cMix server %s (%d/%d)...",
			serverID, i+1, servers.Len())
		_, err := network.SendAskOnline(server)
		if err != nil {
			jww.WARN.Printf("Could not contact cMix server %s (%d/%d): %s...",
				serverID, i+1, servers.Len(), err)
			time.Sleep(250 * time.Millisecond)
		} else {
			i--
		}
	}
}
