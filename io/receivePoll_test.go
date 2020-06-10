////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/elixxir/comms/connect"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network/dataStructures"
	"gitlab.com/elixxir/comms/testkeys"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	ndf2 "gitlab.com/elixxir/primitives/ndf"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/internal/state"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/elixxir/server/testUtil"
	"testing"
	"time"
)

func setupTests(t *testing.T, testState current.Activity) (internal.Instance, *pb.ServerPoll,
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

	ourGateway := internal.GW{
		ID:      &id.TempGateway,
		TlsCert: nil,
		Address: testGatewayAddress,
	}

	// We need to create a server.Definition so we can create a server instance.
	nid := internal.GenerateId(t)
	def := internal.Definition{
		ID:              nid,
		ResourceMonitor: &measure.ResourceMonitor{},
		UserRegistry:    &globals.UserMap{},
		FullNDF:         testNdf,
		PartialNDF:      testNdf,
		Gateway:         ourGateway,
	}

	// Here we create a server instance so that we can test the poll ndf.
	m := state.NewTestMachine(dummyStates, testState, t)

	instance, err := internal.CreateServerInstance(&def, NewImplementation,
		m, "1.1.0")
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
	_, err = instance.GetNetwork().AddHost(&id.Permissioning, "", cert, false, false)
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
		Full:       &pb.NDFHash{Hash: fullHash1},
		Partial:    &pb.NDFHash{Hash: fullHash1},
		LastUpdate: 0,
		Error:      "",
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

	// Create host and auth
	h, _ := connect.NewHost(instance.GetGateway(), testGatewayAddress, nil, false, false)
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          h,
	}

	res, err := ReceivePoll(poll, &instance, testGatewayAddress, auth)
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

	// Create host and auth
	h, _ := connect.NewHost(instance.GetGateway(), testGatewayAddress, nil, false, false)
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          h,
	}

	res, err := ReceivePoll(poll, &instance, testGatewayAddress, auth)
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

	// Create host and auth
	h, _ := connect.NewHost(instance.GetGateway(), testGatewayAddress, nil, false, false)
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          h,
	}

	res, err := ReceivePoll(poll, &instance, testGatewayAddress, auth)
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

	// Create host and auth
	h, _ := connect.NewHost(instance.GetGateway(), testGatewayAddress, nil, false, false)
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          h,
	}

	res, err := ReceivePoll(poll, &instance, testGatewayAddress, auth)
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

	// Create host and auth
	h, _ := connect.NewHost(instance.GetGateway(), testGatewayAddress, nil, false, false)
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          h,
	}

	res, err := ReceivePoll(poll, &instance, testGatewayAddress, auth)
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

	// Create host and auth
	h, _ := connect.NewHost(instance.GetGateway(), testGatewayAddress, nil, false, false)
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          h,
	}

	res, err := ReceivePoll(poll, &instance, testGatewayAddress, auth)
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

	// Create host and auth
	h, _ := connect.NewHost(instance.GetGateway(), testGatewayAddress, nil, false, false)
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          h,
	}

	res, err := ReceivePoll(poll, &instance, testGatewayAddress, auth)
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

	res, err = ReceivePoll(poll, &instance, testGatewayAddress, auth)
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

	// Create host and auth
	h, _ := connect.NewHost(instance.GetGateway(), testGatewayAddress, nil, false, false)
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          h,
	}

	res, err := ReceivePoll(poll, &instance, testGatewayAddress, auth)
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

// ----------------------- Auth Errors ------------------------------------

// Test error case in which sender of ReceivePoll is not authenticated
func TestReceivePoll_Unauthenticated(t *testing.T) {
	instance, pollMsg, _, _ := setupTests(t, current.REALTIME)

	// Create host and auth
	h, _ := connect.NewHost(instance.GetGateway(), testGatewayAddress, nil, false, false)
	auth := &connect.Auth{
		IsAuthenticated: false,
		Sender:          h,
	}

	expectedError := connect.AuthError(auth.Sender.GetId()).Error()

	// Call ReceivePoll with bad auth
	_, err := ReceivePoll(pollMsg, &instance, testGatewayAddress, auth)
	if err.Error() != expectedError {
		t.Errorf("Did not receive expected error!"+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", connect.AuthError(auth.Sender.GetId()), err)
	}
}

// Test error case in which sender of ReceivePoll has an unexpected ID
func TestReceivePoll_Auth_BadId(t *testing.T) {
	instance, pollMsg, _, _ := setupTests(t, current.REALTIME)

	// Set auth with unexpected id
	badGatewayId := id.NewIdFromString("bad", id.Gateway, t)

	// Create host and auth
	h, _ := connect.NewHost(badGatewayId, testGatewayAddress, nil, false, false)
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          h,
	}

	// Reset auth error
	expectedError := connect.AuthError(auth.Sender.GetId()).Error()

	_, err := ReceivePoll(pollMsg, &instance, testGatewayAddress, auth)
	if err.Error() != expectedError {
		t.Errorf("Did not receive expected error!"+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", connect.AuthError(auth.Sender.GetId()), err)
	}

}

// Test multiple poll calls. First call is to set up a happy path
// Second call uses the same parameters, and is expected to fail due to new expectations for auth object
// Third call uses new auth object with expected parameters, expected happy path
func TestReceivePoll_Auth_DoublePoll(t *testing.T) {
	instance, pollMsg, _, _ := setupTests(t, current.REALTIME)

	// Create host and auth
	h, _ := connect.NewHost(instance.GetGateway(), testGatewayAddress, nil, false, false)
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          h,
	}

	// Happy path of 1st receive poll for auth
	_, err := ReceivePoll(pollMsg, &instance, testGatewayAddress, auth)
	if err != nil {
		t.Errorf("Did not receive expected error!"+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", connect.AuthError(auth.Sender.GetId()), err)
	}

	expectedError := connect.AuthError(auth.Sender.GetId()).Error()

	// Attempt the same poll, with parameters that should now be invalid
	_, err = ReceivePoll(pollMsg, &instance, testGatewayAddress, auth)
	if err.Error() != expectedError {
		t.Errorf("Expected happy path, received error: %v", err)
	}

	// Get a copy of the server id and transfer to a gateway id
	newGatewayId := instance.GetID().DeepCopy()
	newGatewayId.SetType(id.Gateway)

	// Create host and auth with new parameters, namely a gateway id based off of the server id
	h, _ = connect.NewHost(newGatewayId, testGatewayAddress, nil, false, false)
	auth = &connect.Auth{
		IsAuthenticated: true,
		Sender:          h,
	}

	// Attempt second poll with new, expected parameters
	_, err = ReceivePoll(pollMsg, &instance, testGatewayAddress, auth)
	if err != nil {
		t.Errorf("Expected happy path, received error: %v", err)
	}

}
