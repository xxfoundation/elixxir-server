////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package receivers

import (
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network/dataStructures"
	"gitlab.com/elixxir/comms/testkeys"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/primitives/id"
	ndf2 "gitlab.com/elixxir/primitives/ndf"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/server/state"
	"gitlab.com/elixxir/server/testUtil"
	"testing"
	"time"
)


// These are ndf hash value we use through out the tests to change the expected out put.
var fullHash1 = []byte("")
var fullHash2 = []byte("")
var partialHash1 = []byte("")
var partialHash2 = []byte("")

func setupTests(t *testing.T) (server.Instance, *pb.ServerPoll){
	//Get a new ndf
	testNdf, _, err := ndf2.DecodeNDF(testUtil.ExampleNDF)
	if err != nil{
		t.Logf("Failed to decode ndf")
		t.Fail()
	}

	// Since no deep copy of ndf exists we create a new object entirely for second ndf that
	// We use to test against
	test2Ndf, _, err := ndf2.DecodeNDF(testUtil.ExampleNDF)
	if err != nil{
		t.Logf("Failed to decode ndf 2")
		t.Fail()
	}

	// Change the time of the ndf so we can generate a different hash for use in comparisons
	test2Ndf.Timestamp = time.Now()
	fullHash2, err = dataStructures.GenerateNDFHash(test2Ndf)
	partialHash2, err = dataStructures.GenerateNDFHash(test2Ndf)


	// We need to create a server.Definition so we can create a server instance.
	nid := server.GenerateId(t)
	def := server.Definition{
		ID:              nid,
		ResourceMonitor: &measure.ResourceMonitor{},
		UserRegistry:    &globals.UserMap{},
		FullNDF: 		 testNdf,
		PartialNDF: 	 testNdf,
	}

	// Here we create a server instance so that we can test the poll ndf.
	m := state.NewMachine(dummyStates)
	instance, err := server.CreateServerInstance(&def, NewImplementation, m, false)
	if err != nil {
		t.Logf("failed to create server Instance")
		t.Fail()
	}

	// In order for our instance to return updated ndf we need to sign it so here we extract keys
	certPath := testkeys.GetGatewayCertPath()
	cert := testkeys.LoadFromPath(certPath)
	keyPath := testkeys.GetGatewayKeyPath()
	privKeyPem := testkeys.LoadFromPath(keyPath)
	privKey, err := rsa.LoadPrivateKeyFromPem(privKeyPem)
	if err != nil{
		t.Logf("Private Key failed to generate %v", err)
		t.Fail()
	}

	// Add the certs to our network instance
	instance.GetNetwork().AddHost(id.PERMISSIONING, "", cert,false, false)

	// Generate and sign the new ndf with the key we retrieved
	f := pb.NDF {}
	f.Ndf, err = testNdf.Marshal()
	if err != nil{
		t.Log(err)
		t.Fail()
	}
	err = signature.Sign(&f, privKey)
	if err != nil{
		t.Log(err)
		t.Fail()
	}

	// Push ndf updates to our instance so we can retrieve them from poll function
	err = instance.GetConsensus().UpdateFullNdf(&f)
	if err != nil{
		t.Log(err)
		t.Fail()
	}

	err = instance.GetConsensus().UpdatePartialNdf(&f)
	if err != nil{
		t.Log(err)
		t.Fail()
	}

	fullHash1 = instance.GetConsensus().GetFullNdf().GetHash()
	partialHash1 = instance.GetConsensus().GetPartialNdf().GetHash()

	//Push a round update that can be used for the test:
	poll := pb.ServerPoll{
		Full:                 &pb.NDFHash{Hash: fullHash1},
		Partial:              &pb.NDFHash{Hash: fullHash1},
		LastUpdate:           0,
		Error:                "",
		XXX_NoUnkeyedLiteral: struct{}{},
		XXX_unrecognized:     nil,
		XXX_sizecache:        0,
	}

	return *instance, &poll

}

// Test what happens when you send in an all nil result.
func TestReceivePoll_NoUpdates(t *testing.T) {

	instance, poll := setupTests(t)

	res, err := ReceivePoll(poll, &instance)
	if err != nil{
		t.Logf("Unexpected error %v", err)
		t.Fail()
	}
	if res == nil{
		t.Logf("Response was nil")
		t.Fail()
	}

	if res.Slots != nil{
		t.Logf("ServerPollResponse.Slots is not nil")
		t.Fail()
	}
	if res.BatchRequest != nil{
		t.Logf("ServerPollResponse.BatchRequest is not nil")
		t.Fail()
	}

	if len(res.Updates) > 0   {
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


// Test that when the partial ndf hash is different as the incoming ndf hash
// the ndf returned in the server poll is the new ndf from the poll
func TestReceivePoll_DifferentFullNDF(t *testing.T) {
	instance, poll := setupTests(t)
	//Change the full hash so we get a the new ndf returned to us
	poll.Full.Hash = fullHash2

	res, err := ReceivePoll(poll, &instance)
	if err != nil {
		t.Logf("Unexpected error %v", err)
		t.Fail()
	}

	if (res.FullNDF == nil) {
		t.Logf("ReceivePoll should have returned a new ndf")
		t.Fail()
	}
}

// Test that when the fulll ndf hash is the same as the
// incomming ndf hash the ndf returned in the server poll is the same ndf we started out withfunc TestRecievePoll_SameFullNDF(t *testing.T) {
func TestReceivePoll_SameFullNDF(t *testing.T) {
	instance, poll := setupTests(t)
	res, err := ReceivePoll(poll, &instance)
	if err != nil {
		t.Logf("Unexpected error %v", err)
		t.Fail()
	}

	if (res.FullNDF != nil) {
		t.Logf("ReceivePoll should have not returned the same ndf from instance")
		t.Fail()
	}
}

// Test that when the partial ndf hash is different as the incoming ndf hash
// the ndf returned in the server poll is the new ndf from the poll
func TestReceivePoll_DifferentPartiallNDF(t *testing.T) {
	instance, poll := setupTests(t)
	poll.Partial.Hash = fullHash2

	res, err := ReceivePoll(poll, &instance)
	if err != nil {
		t.Logf("Unexpected error %v", err)
		t.Fail()
	}

	if (res.PartialNDF == nil) {
		t.Logf("ReceivePoll should have returned a new ndf")
		t.Fail()
	}
}

// Test that when the partial ndf hash is the same as the
// incoming ndf hash the ndf returned in the server poll is the same ndf we started out with
func TestReceivePoll_SamePartialNDF(t *testing.T) {
	instance, poll := setupTests(t)
	res, err := ReceivePoll(poll, &instance)
	if err != nil {
		t.Logf("Unexpected error %v", err)
		t.Fail()
	}

	if (res.PartialNDF != nil) {
		t.Logf("ReceivePoll should not have returned a new ndf: %v", res.PartialNDF)
		t.Fail()
	}
}

// Send a round update to receive poll and test that we get the expected value back in server poll response
func TestReceivePoll_GetRoundUpdates(t *testing.T) {
	instance, poll := setupTests(t)
	newRound := &pb.RoundInfo{
		ID: uint64(23),
	}

	instance.GetConsensus().RoundUpdate(newRound)
	res, err := ReceivePoll(poll, &instance)
	if err != nil {
		t.Logf("Unexpected error %v", err)
		t.Fail()
	}

	if (res.Updates[0].ID != 23) {
		t.Logf("We did not recieve the expected round id.")
		t.Fail()
	}
}

// Send a batch message to receive poll function and test that the returned value is expected
func TestReceivePoll_GetBatchMessage(t *testing.T) {

}
