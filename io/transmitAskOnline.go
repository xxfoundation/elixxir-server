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
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/xx_network/comms/connect"
	"sync/atomic"
	"time"
)

// VerifyServersOnline Blocks until all given servers respond
func VerifyServersOnline(network *node.Comms, servers *connect.Circuit,
	timeoutDuration time.Duration) error {
	var stop uint64
	finished := make(chan bool, servers.Len())

	// This helper runs until successfully connected or stop is not 0
	askOnline := func(i int) {
		// Pull server's host from the connection manager
		serverID := servers.GetNodeAtIndex(i)
		server := servers.GetHostAtIndex(i)

		// Send AskOnline to all servers
		jww.INFO.Printf("Waiting for cMix server %s (%d/%d)...",
			serverID, i+1, servers.Len())
		for atomic.LoadUint64(&stop) == 0 {
			_, err := network.SendAskOnline(server)
			if err != nil {
				jww.WARN.Printf("Could not contact "+
					"cMix server %s (%d/%d): %+v",
					serverID, i+1, servers.Len(),
					err)
				time.Sleep(250 * time.Millisecond)
				continue
			}
			jww.INFO.Printf("cMix server %s (%d/%d) "+
				"is online...",
				serverID, i+1, servers.Len())
			finished <- true
			return
		}
	}

	// Start goroutines to check liveness
	for i := 0; i < servers.Len(); i++ {
		go askOnline(i)
	}

	// Handle timeout and error reporting
	var err error
	err = nil
	finishedCnt := 0
	// We are done when all servers responded
	for finishedCnt < servers.Len() && err == nil {
		select {
		case <-finished:
			finishedCnt++
			break
		case <-time.After(timeoutDuration):
			err = errors.Errorf("Timed out connecting to nodes")
			break
		}
	}

	// Close all goroutines
	atomic.AddUint64(&stop, 1)
	return err
}
