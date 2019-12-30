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
	"gitlab.com/elixxir/primitives/id"
	"time"
)

// VerifyServersOnline Blocks until all given servers respond
func VerifyServersOnline(network *node.Comms, servers *connect.Circuit, id *id.Node) {
	for i := 0; i < servers.Len(); {
		// Pull server's host from the connection manager
		serverID := servers.GetNodeAtIndex(i)
		server := servers.GetHostAtIndex(i)
		if serverID.String() == id.String() {
			i++
			continue
		}

		// Send comm to the other server
		_, err := network.SendAskOnline(server)
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
