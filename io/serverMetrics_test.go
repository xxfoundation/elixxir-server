package io

import (
	"gitlab.com/privategrity/comms/node"
	"gitlab.com/privategrity/server/globals"
	"testing"
)

func TestServerMetrics(t *testing.T) {
	servers := []string{"localhost:11420", "localhost:11421", "localhost:11422"}
	// Initialize GlobalRoundMap
	globals.GlobalRoundMap = globals.NewRoundMap()
	// Kick off comms server
	for i := range servers {
		go node.StartServer(servers[i],
			ServerImpl{Rounds: &globals.GlobalRoundMap}, "", "")
		if i == len(servers)-1 {
			NextServer = servers[0]
		} else {
			NextServer = servers[i+1]
		}
	}
	GetServerMetrics(servers)
}
