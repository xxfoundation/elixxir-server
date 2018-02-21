////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/privategrity/comms/mixserver"
	"gitlab.com/privategrity/server/globals"
	"os"
	"testing"
)

// Start server for testing
func TestMain(m *testing.M) {
	// Initialize GlobalRoundMap
	globals.GlobalRoundMap = globals.NewRoundMap()
	// Kick off comms server
	localServer := "localhost:5555"
	go mixserver.StartServer(localServer,
		ServerImpl{Rounds: &globals.GlobalRoundMap})
	// Next hop will be back to us
	NextServer = localServer
	os.Exit(m.Run())
}
