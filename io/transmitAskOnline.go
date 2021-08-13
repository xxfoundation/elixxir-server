///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package io

// transmitAskOnline.go contains the logic for transmitting an askOnline comm

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"git.xx.network/elixxir/comms/node"
	"git.xx.network/xx_network/comms/connect"
	"git.xx.network/xx_network/primitives/id"
	"time"
)

// VerifyServersOnline Blocks until all given servers respond
func VerifyServersOnline(network *node.Comms, servers *connect.Circuit,
	timeoutDuration time.Duration) error {
	finished := make(chan int, servers.Len())

	// This helper runs until successfully connected or stop is not 0
	askOnline := func(i int) {
		// Pull server's host from the connection manager
		serverID := servers.GetNodeAtIndex(i)
		server := servers.GetHostAtIndex(i)

		// Send AskOnline to all servers
		jww.INFO.Printf("Waiting for cMix server %s (%d/%d)...",
			serverID, i+1, servers.Len())
		_, err := network.SendAskOnline(server)
		if err != nil {
			jww.WARN.Printf("Could not contact "+
				"cMix server %s (%d/%d): %+v",
				serverID, i+1, servers.Len(),
				err)
			// We don't report we finished because we
			// error'd out and want to fail the function.
		} else {
			jww.INFO.Printf("cMix server %s (%d/%d) "+
				"is online...",
				serverID, i+1, servers.Len())
			finished <- i
		}
	}

	// Start goroutines to check liveness
	for i := 0; i < servers.Len(); i++ {
		go askOnline(i)
	}

	// Handle timeout and error reporting
	var err error
	err = nil
	finishedNodes := make([]bool, servers.Len())
	finishedCnt := 0
	// We are done when all servers responded
	for finishedCnt < servers.Len() && err == nil {
		select {
		case i := <-finished:
			finishedNodes[i] = true
			finishedCnt++
			break
		case <-time.After(timeoutDuration):
			var unfinished []*id.ID
			for i := 0; i < servers.Len(); i++ {
				if finishedNodes[i] != true {
					unfinished = append(unfinished,
						servers.GetNodeAtIndex(i))
				}
			}
			err = errors.Errorf("Timed out connecting to nodes: "+
				"%+v", unfinished)
			break
		}
	}

	return err
}
