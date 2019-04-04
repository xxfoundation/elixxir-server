////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/server/globals"
	"os"
	"testing"
)

// Start server for testing
func TestMain(m *testing.M) {
	// Initialize GlobalRoundMap
	globals.GlobalRoundMap = globals.NewRoundMap()
	// Kick off comms server
	localServer := "localhost:5555"
	go node.StartServer(localServer,
		NewServerImplementation(), "", "")
	// Next hop will be back to us
	NextServer = localServer
	os.Exit(m.Run())
}

func TestServerImpl_SetPublicKey(t *testing.T) {
	roundId := "test"
	expected := globals.GetGroup().NewInt(5)
	globals.GlobalRoundMap = globals.NewRoundMap()
	globals.GlobalRoundMap.AddRound(roundId,
		globals.NewRound(5, globals.GetGroup()))

	SetPublicKey(roundId, expected.Bytes())

	actual := globals.GlobalRoundMap.GetRound(roundId).CypherPublicKey
	if actual.Cmp(expected) != 0 {
		t.Errorf("SetPublicKey: Values did not match!"+
			" Expected %s Actual %s", expected.Text(10),
			actual.Text(10))
	}
}
