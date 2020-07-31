///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package io

// transmitAskOnline.go contains the logic for transmitting an askOnline comm

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/xx_network/comms/connect"
	"time"
)

// VerifyServersOnline Blocks until all given servers respond
func VerifyServersOnline(network *node.Comms, servers *connect.Circuit) {
	for i := 0; i < servers.Len(); {
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
			i++
			jww.INFO.Printf("cMix server %s (%d/%d) is online...",
				serverID, i+1, servers.Len())
		}
	}
}
