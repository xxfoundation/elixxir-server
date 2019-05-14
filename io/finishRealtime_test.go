////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server/round"
	"testing"
	"time"
)

var pString = "9DB6FB5951B66BB6FE1E140F1D2CE5502374161FD6538DF1648218642F0B5C48" +
	"C8F7A41AADFA187324B87674FA1822B00F1ECF8136943D7C55757264E5A1A44F" +
	"FE012E9936E00C1D3E9310B01C7D179805D3058B2A9F4BB6F9716BFE6117C6B5" +
	"B3CC4D9BE341104AD4A80AD6C94E005F4B993E14F091EB51743BF33050C38DE2" +
	"35567E1B34C3D6A5C0CEAA1A0F368213C3D19843D0B4B09DCB9FC72D39C8DE41" +
	"F1BF14D4BB4563CA28371621CAD3324B6A2D392145BEBFAC748805236F5CA2FE" +
	"92B871CD8F9C36D3292B5509CA8CAA77A2ADFC7BFD77DDA6F71125A7456FEA15" +
	"3E433256A2261C6A06ED3693797E7995FAD5AABBCFBE3EDA2741E375404AE25B"

var gString = "5C7FF6B06F8F143FE8288433493E4769C4D988ACE5BE25A0E24809670716C613" +
	"D7B0CEE6932F8FAA7C44D2CB24523DA53FBE4F6EC3595892D1AA58C4328A06C4" +
	"6A15662E7EAA703A1DECF8BBB2D05DBE2EB956C142A338661D10461C0D135472" +
	"085057F3494309FFA73C611F78B32ADBB5740C361C9F35BE90997DB2014E2EF5" +
	"AA61782F52ABEB8BD6432C4DD097BC5423B285DAFB60DC364E8161F4A2A35ACA" +
	"3A10B1C4D203CC76A470A33AFDCBDD92959859ABD8B56E1725252D78EAC66E71" +
	"BA9AE3F1DD2487199874393CD4D832186800654760E1E34C09E4D155179F9EC0" +
	"DC4473F996BDCE6EED1CABED8B6F116F7AD9CF505DF0F998E34AB27514B0FFE7"

var qString = "F2C3119374CE76C9356990B465374A17F23F9ED35089BD969F61C6DDE9998C1F"

var p = large.NewIntFromString(pString, 16)
var g = large.NewIntFromString(gString, 16)
var q = large.NewIntFromString(qString, 16)

var grp = cyclic.NewGroup(p, g, q)

var received = make(chan *mixmessages.RoundInfo, 100)

func MockFinishRealtimeImplementation() *node.Implementation {
	impl := node.NewImplementation()
	impl.Functions.FinishRealtime = func(message *mixmessages.RoundInfo) error {
		received <- message
		return nil
	}
	return impl
}

// Test that SendFinishRealtime correctly broadcasts message
// to all other nodes
func TestSendFinishRealtime(t *testing.T) {
	//Setup the network
	numNodes := 4
	numRecv := 0
	comms, topology := buildTestNetworkComponents(
		[]func() *node.Implementation{
			MockFinishRealtimeImplementation,
			MockFinishRealtimeImplementation,
			MockFinishRealtimeImplementation,
			MockFinishRealtimeImplementation,
			MockFinishRealtimeImplementation,})
	defer Shutdown(comms)

	selfID := topology.GetNodeAtIndex(0)
	rndID := id.Round(42)
	err := SendFinishRealtime(comms[0], rndID, topology, selfID)

	if err != nil {
		t.Errorf("SendFinishRealtime: Unexpected error: %+v", err)
	}

LOOP:
	for {
		select {
		case msg:= <-received:
			if id.Round(msg.ID) != rndID {
				t.Errorf("SendFinishRealtime: Incorrect round ID"+
					"Expected: %v, Recieved: %v", rndID, msg.ID)
			}
			numRecv++
			if numRecv == numNodes {
				break LOOP
			}
		case <-time.After(5*time.Second):
			t.Errorf("Test timed out!")
			break LOOP
		}
	}
}

// Test that FinishRealtime correctly handles reception of finish realtime
// message, by deleting the round from round manager
// Confirm that the function errors out when the round doesn't exist
func TestFinishRealtime(t *testing.T) {
	rm := round.NewManager()
	roundID := id.Round(42)

	topology := circuit.New([]*id.Node{&id.Node{}})

	round := round.New(grp, roundID, nil, nil, topology,
		&id.Node{}, 5)

	rm.AddRound(round)

	msg := &mixmessages.RoundInfo{ID: uint64(roundID)}

	err := FinishRealtime(rm, msg)

	if err != nil {
		t.Errorf("FinishRealtime: Unexpected error: %+v", err)
	}

	err = FinishRealtime(rm, msg)

	if err == nil {
		t.Errorf("FinishRealtime: Should have returned error")
	}
}
