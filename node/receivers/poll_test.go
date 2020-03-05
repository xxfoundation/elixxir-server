////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////


package receivers

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/node"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/server/state"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"testing"
)

// Total of 7 tests needed
// test recieve nothing at all
// test recieve everything
// test recieved just each individual part

func testSetup(t *testing.T) (server.Instance, *mixmessages.ServerPoll){
	primeString := "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
		"29024E088A67CC74020BBEA63B139B22514A08798E3404DD" +
		"EF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245" +
		"E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7ED" +
		"EE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3D" +
		"C2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F" +
		"83655D23DCA3AD961C62F356208552BB9ED529077096966D" +
		"670C354E4ABC9804F1746C08CA18217C32905E462E36CE3B" +
		"E39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9" +
		"DE2BCBF6955817183995497CEA956AE515D2261898FA0510" +
		"15728E5A8AACAA68FFFFFFFFFFFFFFFF"
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2))

	//Generate everything needed to make a user
	nid := server.GenerateId(t)
	def := server.Definition{
		ID:              nid,
		CmixGroup:       grp,
		Topology:        connect.NewCircuit([]*id.Node{nid}),
		ResourceMonitor: &measure.ResourceMonitor{},
		UserRegistry:    &globals.UserMap{},
	}

	changeList := [current.NUM_STATES]state.Change{}

	instance, err := server.CreateServerInstance(&def, NewImplementation, changeList,false)
	if err != nil{
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
	if err == nil{
		t.Logf("Unexpected error %v", err)
		t.Fail()
	}

	if( res.Slots != nil){
		t.Logf("ServerPollResponse.Slots is not nil")
		t.Fail()
	}
	if( res.BatchRequest != nil){
		t.Logf("ServerPollResponse.BatchRequest is not nil")
		t.Fail()
	}
	if( res.Updates != nil){
		t.Logf("ServerPollResponse.Updates is not nil")
		t.Fail()
	}
	if( res.Id != nil){
		t.Logf("ServerPollResponse.Id is not nil")
		t.Fail()
	}
	if( res.FullNDF != nil){
		t.Logf("ServerPollResponse.ul is not nil")
		t.Fail()
	}
}

// NewImplementation creates a new implementation of the server.
// When a function is added to comms, you'll need to point to it here.
func NewImplementation(instance *server.Instance) *node.Implementation {

	impl := node.NewImplementation(instance)

	return impl
}
