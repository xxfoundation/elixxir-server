package io

import (
	"testing"
	"gitlab.com/privategrity/comms/node"
	"gitlab.com/privategrity/server/globals"
)

func TestGetRoundtripPing(t *testing.T) {
	servers := []string{"localhost:50000", "localhost:50001", "localhost:50002"}
	// Initialize GlobalRoundMap
	globals.GlobalRoundMap = globals.NewRoundMap()
	// Kick off comms server
	for i := range servers {
		go node.StartServer(servers[i],
			ServerImpl{Rounds: &globals.GlobalRoundMap})
		if i == len(servers)-1 {
			NextServer = servers[0]
		} else {
			NextServer = servers[i+1]
		}
	}
	GetRoundtripPing(servers)
}

/*func TestGetRoundtripPing_OneServer(t *testing.T) {
	server := []string{"localhost:50000"}
	// Initialize GlobalRoundMap
	globals.GlobalRoundMap = globals.NewRoundMap()
	// Kick off comms server
	go node.StartServer(server[0], ServerImpl{Rounds: &globals.GlobalRoundMap})
	NextServer = server[0]
	GetRoundtripPing(server)
}*/
