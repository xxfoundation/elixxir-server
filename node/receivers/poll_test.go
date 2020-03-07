////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package receivers

import (
	"fmt"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/testkeys"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
	ndf2 "gitlab.com/elixxir/primitives/ndf"
	"gitlab.com/elixxir/primitives/utils"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/server/state"
	"math/rand"
	"testing"
	"time"
)

// Total of 7 tests needed
// test recieve nothing at all
// test recieve everything
// test recieved just each individual part
var fullHash1 = []byte("")
var fullHash2 = []byte("")
var partialHash1 = []byte("")
var partialHash2 = []byte("")

func setupTests(t *testing.T) (server.Instance, *mixmessages.ServerPoll){

	nodeId := id.NewNodeFromUInt(uint64(0), t)
	nodeAddr := fmt.Sprintf("0.0.0.0:%d", 6000+rand.Intn(1000))
	cert, _ := utils.ReadFile(testkeys.GetNodeCertPath())
	gAddr := fmt.Sprintf("0.0.0.0:%d", 5000+rand.Intn(1000))

	//Generate everything needed to make a user
	node := ndf.Node{
		ID:             nodeId.Bytes(),
		TlsCertificate: string(cert),
		Address:        nodeAddr,
	}
	gw := ndf.Gateway{
		Address:        gAddr,
		TlsCertificate: string(cert),
	}
	testNdf := &ndf2.NetworkDefinition{
		Timestamp: time.Now(),
		Nodes:     []ndf.Node{node},
		Gateways:  []ndf.Gateway{gw},
		E2E:       ndf.Group{},
		CMIX:      ndf.Group{},
		UDB:       ndf.UDB{},
	}

	nid := server.GenerateId(t)
	def := server.Definition{
		ID:              nid,
		ResourceMonitor: &measure.ResourceMonitor{},
		UserRegistry:    &globals.UserMap{},
		NDF: 			 testNdf,
	}

	m := state.NewMachine(dummyStates)
	instance, err := server.CreateServerInstance(&def, NewImplementation, m, false)
	if err != nil {
		t.Logf("failed to create server Instance")
		t.Fail()
	}

	poll := mixmessages.ServerPoll{
		Full:                 &mixmessages.NDFHash{Hash: fullHash1},
		Partial:              &mixmessages.NDFHash{Hash: partialHash1},
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

	instance, poll := setupTests(t)

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

// Test what happens when you send in an all nil result.
func TestRecievePoll_RoundUpdatesFail(t *testing.T) {
	instance, poll := setupTests(t)
	res, err := RecievePoll(poll, &instance)
	if err != nil{
		t.Logf("Round updates should have failed")
		t.Fail()
	}

	if res != nil{
		t.Logf("Res should return as nil when err is returned")
		t.Fail()
	}
}


// Test that when the partial ndf hash is different as the incomming ndf hash
// the ndf returned in the server poll is the new ndf from the poll
func TestRecievePoll_DifferentFullNDF(t *testing.T) {
	instance, poll := setupTests(t)
	res, err := RecievePoll(poll, &instance)
	if err == nil {
		t.Logf("Unexpected error %v", err)
		t.Fail()
	}

	if (res.FullNDF != instance.GetConsensus().GetFullNdf().GetPb()) {
		t.Logf("ReceivePoll should have not returned a new ndf")
		t.Fail()
	}
}

// Test that when the fulll ndf hash is the same as the
// incomming ndf hash the ndf returned in the server poll is the same ndf we started out withfunc TestRecievePoll_SameFullNDF(t *testing.T) {
func TestRecievePoll_SameFullNDF(t *testing.T) {
	instance, poll := setupTests(t)
	//poll.Full = fullHash2
	res, err := RecievePoll(poll, &instance)
	if err == nil {
		t.Logf("Unexpected error %v", err)
		t.Fail()
	}

	if (res.FullNDF == instance.GetConsensus().GetFullNdf().GetPb()) {
		t.Logf("ReceivePoll should have returned a new ndf")
		t.Fail()
	}
}

// Test that when the partial ndf hash is different as the incomming ndf hash
// the ndf returned in the server poll is the new ndf from the poll
func TestRecievePoll_DifferentPartiallNDF(t *testing.T) {
	instance, poll := setupTests(t)
	res, err := RecievePoll(poll, &instance)
	if err == nil {
		t.Logf("Unexpected error %v", err)
		t.Fail()
	}

	if (res.FullNDF != instance.GetConsensus().GetFullNdf().GetPb()) {
		t.Logf("ReceivePoll should have not returned a new ndf")
		t.Fail()
	}
}

// Test that when the partial ndf hash is the same as the
// incomming ndf hash the ndf returned in the server poll is the same ndf we started out with
func TestRecievePoll_SamePartialNDF(t *testing.T) {
	instance, poll := setupTests(t)
	//poll.Full = fullHash2
	res, err := RecievePoll(poll, &instance)
	if err == nil {
		t.Logf("Unexpected error %v", err)
		t.Fail()
	}

	if (res.FullNDF == instance.GetConsensus().GetFullNdf().GetPb()) {
		t.Logf("ReceivePoll should have returned a new ndf")
		t.Fail()
	}
}