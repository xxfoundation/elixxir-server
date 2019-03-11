////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/server/globals"
	"testing"
)

func TestServerMetrics(t *testing.T) {
	servers := []string{"localhost:11420", "localhost:11421", "localhost:11422"}
	// Initialize GlobalRoundMap
	globals.GlobalRoundMap = globals.NewRoundMap()
	// Kick off comms server
	for i := range servers {
		go node.StartServer(servers[i],
			NewServerImplementation(), "", "")
		if i == len(servers)-1 {
			NextServer = servers[0]
		} else {
			NextServer = servers[i+1]
		}
	}
	GetServerMetrics(servers)
}
