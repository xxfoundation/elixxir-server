////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package receivers

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/server/state"
	"gitlab.com/elixxir/server/testUtil"
	"testing"
)

// Total of 7 tests needed
// test recieve nothing at all
// test recieve everything
// test recieved just each individual part

func testSetup(t *testing.T) (server.Instance, *mixmessages.ServerPoll) {
	//Generate everything needed to make a user
	nid := server.GenerateId(t)
	def := server.Definition{
		ID:              nid,
		ResourceMonitor: &measure.ResourceMonitor{},
		UserRegistry:    &globals.UserMap{},
		FullNDF:         testUtil.NDF,
		PartialNDF:      testUtil.NDF,
	}

	m := state.NewMachine(dummyStates)
	instance, err := server.CreateServerInstance(&def, NewImplementation, m, false)
	if err != nil {
		t.Logf("failed to create server Instance")
		t.Fail()
	}

	poll := mixmessages.ServerPoll{
		Full:                 nil,
		Partial:              nil,
		LastUpdate:           0,
		Error:                "",
		XXX_NoUnkeyedLiteral: struct{}{},
		XXX_unrecognized:     nil,
		XXX_sizecache:        0,
	}

	return *instance, &poll
}

// Test what happens when you send in an all nil result.
func TestRecievePoll_AllNil(t *testing.T) {

	instance, poll := testSetup(t)

	res, err := RecievePoll(poll, &instance)
	if err != nil {
		t.Logf("Unexpected error %v", err)
		t.Fail()
	}

	if res.Slots != nil {
		t.Logf("ServerPollResponse.Slots is not nil")
		t.Fail()
	}
	if res.BatchRequest != nil {
		t.Logf("ServerPollResponse.BatchRequest is not nil")
		t.Fail()
	}
	if res.Updates != nil {
		t.Logf("ServerPollResponse.Updates is not nil")
		t.Fail()
	}
	if res.Id != nil {
		t.Logf("ServerPollResponse.Id is not nil")
		t.Fail()
	}
	if res.FullNDF != nil {
		t.Logf("ServerPollResponse.ul is not nil")
		t.Fail()
	}
}
