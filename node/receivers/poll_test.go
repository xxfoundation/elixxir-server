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
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	ndf2 "gitlab.com/elixxir/primitives/ndf"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/server/state"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/elixxir/server/testUtil"
	"testing"
	"time"
)

func setupTests(t *testing.T, test_state current.Activity) (server.Instance, *pb.ServerPoll,
	[]byte, *rsa.PrivateKey) {
	//Get a new ndf
	testNdf, _, err := ndf2.DecodeNDF(testUtil.ExampleNDF)
	if err != nil {
		t.Logf("Failed to decode ndf")
		t.Fail()
	}

	// Since no deep copy of ndf exists we create a new object entirely for second ndf that
	// We use to test against
	test2Ndf, _, err := ndf2.DecodeNDF(testUtil.ExampleNDF)
	if err != nil {
		t.Logf("Failed to decode ndf 2")
		t.Fail()
	}

	// Change the time of the ndf so we can generate a different hash for use in comparisons
	test2Ndf.Timestamp = time.Now()

	// We need to create a server.Definition so we can create a server instance.
	nid := server.GenerateId(t)
	def := server.Definition{
		ID:              nid,
		ResourceMonitor: &measure.ResourceMonitor{},
		UserRegistry:    &globals.UserMap{},
		FullNDF:         testNdf,
		PartialNDF:      testNdf,
	}

	// Here we create a server instance so that we can test the poll ndf.
	m := state.NewTestMachine(dummyStates, test_state, t)

	instance, err := server.CreateServerInstance(&def, NewImplementation, m, false)
	if err != nil {
		t.Logf("failed to create server Instance")
		t.Fail()
	}

	//Make sure instance is ready by default
	instance.SetGatewayAsReady()

	// In order for our instance to return updated ndf we need to sign it so here we extract keys
	certPath := testkeys.GetGatewayCertPath()
	cert := testkeys.LoadFromPath(certPath)
	keyPath := testkeys.GetGatewayKeyPath()
	privKeyPem := testkeys.LoadFromPath(keyPath)
	privKey, err := rsa.LoadPrivateKeyFromPem(privKeyPem)
	if err != nil {
		t.Logf("Private Key failed to generate %v", err)
		t.Fail()
	}

	// Add the certs to our network instance
	_, err = instance.GetNetwork().AddHost(id.PERMISSIONING, "", cert, false, false)
	if err != nil {
		t.Logf("Failed to create host, %v", err)
		t.Fail()
	}
	// Generate and sign the new ndf with the key we retrieved
	f := pb.NDF{}
	f.Ndf, err = testNdf.Marshal()
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	err = signature.Sign(&f, privKey)
	if err != nil {
		t.Log(err)
		t.Fail()
	}

	// Push ndf updates to our instance so we can retrieve them from poll function
	err = instance.GetConsensus().UpdateFullNdf(&f)
	if err != nil {
		t.Log(err)
		t.Fail()
	}

	err = instance.GetConsensus().UpdatePartialNdf(&f)
	if err != nil {
		t.Log(err)
		t.Fail()
	}

	fullHash1 := instance.GetConsensus().GetFullNdf().GetHash()

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

	fullHash2, err := dataStructures.GenerateNDFHash(test2Ndf)
	if err != nil {
		t.Logf("error generating hash for 2nd ndf")
		t.Fail()
	}

	return *instance, &poll, fullHash2, privKey

}

// Test what happens when you send in an all nil result.
func TestReceivePoll_NoUpdates(t *testing.T) {

	instance, poll, _, _ := setupTests(t, current.REALTIME)

	dr := round.NewDummyRound(0, 10, t)
	instance.GetRoundManager().AddRound(dr)

	recv := make(chan services.Chunk)

	go func() {
		time.Sleep(2 * time.Second)
		close(recv)
	}()

	res, err := ReceivePoll(poll, &instance)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		t.Fail()
	}
	if res == nil {
		t.Errorf("Response was nil")
		t.Fail()
	}

	if res.Slots != nil {
		t.Errorf("ServerPollResponse.Slots is not nil")
		t.Fail()
	}
	if res.BatchRequest != nil {
		t.Errorf("ServerPollResponse.BatchRequest is not nil")
		t.Fail()
	}

	if len(res.Updates) > 0 {
		t.Logf("ServerPollResponse.Updates is not nil")
		t.Fail()
	}

	if res.FullNDF != nil {
		t.Errorf("ServerPollResponse.ul is not nil")
		t.Fail()
	}
}

// Test that when the partial ndf hash is different as the incoming ndf hash
// the ndf returned in the server poll is the new ndf from the poll
func TestReceivePoll_DifferentFullNDF(t *testing.T) {
	instance, poll, fullHash2, _ := setupTests(t, current.REALTIME)
	//Change the full hash so we get a the new ndf returned to us
	poll.Full.Hash = fullHash2

	res, err := ReceivePoll(poll, &instance)
	if err != nil {
		t.Logf("Unexpected error %v", err)
		t.Fail()
	}

	if res.FullNDF == nil {
		t.Logf("ReceivePoll should have returned a new ndf")
		t.Fail()
	}
}

// Test that when the fulll ndf hash is the same as the
// incomming ndf hash the ndf returned in the server poll is the same ndf we started out withfunc TestRecievePoll_SameFullNDF(t *testing.T) {
func TestReceivePoll_SameFullNDF(t *testing.T) {
	instance, poll, _, _ := setupTests(t, current.REALTIME)
	res, err := ReceivePoll(poll, &instance)
	if err != nil {
		t.Logf("Unexpected error %v", err)
		t.Fail()
	}

	if res.FullNDF != nil {
		t.Logf("ReceivePoll should have not returned the same ndf from instance")
		t.Fail()
	}
}

// Test that when the partial ndf hash is different as the incoming ndf hash
// the ndf returned in the server poll is the new ndf from the poll
func TestReceivePoll_DifferentPartiallNDF(t *testing.T) {
	instance, poll, fullHash2, _ := setupTests(t, current.REALTIME)
	poll.Partial.Hash = fullHash2

	res, err := ReceivePoll(poll, &instance)
	if err != nil {
		t.Logf("Unexpected error %v", err)
		t.Fail()
	}

	if res.PartialNDF == nil {
		t.Logf("ReceivePoll should have returned a new ndf")
		t.Fail()
	}
}

// Test that when the partial ndf hash is the same as the
// incoming ndf hash the ndf returned in the server poll is the same ndf we started out with
func TestReceivePoll_SamePartialNDF(t *testing.T) {
	instance, poll, _, _ := setupTests(t, current.REALTIME)
	res, err := ReceivePoll(poll, &instance)
	if err != nil {
		t.Logf("Unexpected error %v", err)
		t.Fail()
	}

	if res.PartialNDF != nil {
		t.Logf("ReceivePoll should not have returned a new ndf: %v", res.PartialNDF)
		t.Fail()
	}
}

// Send a round update to receive poll and test that we get the expected value back in server poll response
func TestReceivePoll_GetRoundUpdates(t *testing.T) {
	instance, poll, _, privKey := setupTests(t, current.REALTIME)

	PushNRoundUpdates(10, instance, privKey, t)

	res, err := ReceivePoll(poll, &instance)
	if err != nil {
		t.Logf("Unexpected error: %v", err)
		t.Fail()
	}

	t.Logf("ROUND Updates: %v", res.Updates)
	if len(res.Updates) == 0 {
		t.Logf("We did not recieve any updates")
		t.Fail()
	}

	if res.Updates[0].ID != 23 {

	}

	for k := uint64(0); k < 10; k++ {
		if res.Updates[k].ID != k+1 {
			t.Logf("Receive %v instead of the expected round id from round at index %v", res.Updates[k].ID, k)
			t.Fail()
		}
	}
}

// Send a batch request to receive poll function and test that the returned value is expected
func TestReceivePoll_GetBatchRequest(t *testing.T) {
	//show if its not in real time it doesnt get anything
	instance, poll, _, _ := setupTests(t, current.COMPLETED)
	newRound := &pb.RoundInfo{
		ID: uint64(23),
	}
	err := instance.GetRequestNewBatchQueue().Send(newRound)
	if err != nil {
		t.Logf("Failed to send roundInfo to que %v", err)
	}

	res, err := ReceivePoll(poll, &instance)
	if err != nil {
		t.Logf("Unexpected error %v", err)
		t.Fail()
	}

	//show if its in real time it gets the request
	instance, poll, _, _ = setupTests(t, current.REALTIME)
	newRound = &pb.RoundInfo{
		ID: uint64(23),
	}
	err = instance.GetRequestNewBatchQueue().Send(newRound)
	if err != nil {
		t.Logf("Failed to send roundInfo to que %v", err)
	}

	res, err = ReceivePoll(poll, &instance)
	if err != nil {
		t.Logf("Unexpected error %v", err)
		t.Fail()
	}

	if res.GetBatchRequest().ID != newRound.ID {
		t.Logf("Wrong batch request recieved")
		t.Fail()
	}

}

// Send a batch message to receive poll function and test that the returned value is expected
func TestReceivePoll_GetBatchMessage(t *testing.T) {
	instance, poll, _, _ := setupTests(t, current.REALTIME)

	keyPath := testkeys.GetGatewayKeyPath()
	privKeyPem := testkeys.LoadFromPath(keyPath)
	privKey, err := rsa.LoadPrivateKeyFromPem(privKeyPem)
	if err != nil {
		t.Logf("Private Key failed to generate %v", err)
		t.Fail()
	}

	newRound := &pb.RoundInfo{
		ID: uint64(23),
	}

	err = signature.Sign(newRound, privKey)
	if err != nil {
		t.Logf("Could not sign RoundInfo: %v", err)
		t.Fail()
	}

	err = instance.GetConsensus().RoundUpdate(newRound)
	if err != nil {
		t.Errorf("Round update failed: %s", err)
	}

	dr := round.NewDummyRound(23, 10, t)
	instance.GetRoundManager().AddRound(dr)
	cr := round.CompletedRound{
		RoundID: id.Round(23),
		Round:   make([]*pb.Slot, 10),
	}

	for i := 0; i < 10; i++ {
		cr.Round[i] = &pb.Slot{Index: uint32(i)}
	}

	err = instance.GetCompletedBatchQueue().Send(&cr)
	if err != nil {
		t.Logf("We failed to send a completed batch: %v", err)
	}

	res, err := ReceivePoll(poll, &instance)
	if err != nil {
		t.Logf("Unexpected error %v", err)
		t.Fail()
	}

	if len(res.Slots) != 10 {
		t.Logf("We did not recieve the expected amount of slots")
		t.Fail()
	}

	for k := uint32(0); k < 10; k++ {
		if res.Slots[k].Index != k {
			t.Logf("Slots did not match expected index")
		}
	}

}
