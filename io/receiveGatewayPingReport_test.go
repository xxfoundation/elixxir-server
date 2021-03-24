///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package io

import (
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/testkeys"
	"gitlab.com/elixxir/comms/testutils"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/utils"
	"testing"
)

// Smoke test
func TestReceiveGatewayPingReport(t *testing.T) {
	instance, _ := mockInstance(t, mockSharePhaseImpl)

	// Add the certs to our network instance
	cert, _ := utils.ReadFile(testkeys.GetNodeCertPath())
	_, err := instance.GetNetwork().AddHost(&id.Permissioning, "", cert, connect.GetDefaultHostParams())
	if err != nil {
		t.Logf("Failed to create host, %v", err)
		t.Fail()
	}

	// Add to consensus
	roundId := uint64(7)
	newRoundInfo := &pb.RoundInfo{
		ID: uint64(roundId),
	}

	// Mocking permissioning server signing message
	err = testutils.SignRoundInfo(newRoundInfo, t)
	if err != nil {
		t.Errorf("failed to sign round info: %v", err)
	}

	err = instance.GetConsensus().RoundUpdate(newRoundInfo)
	if err != nil {
		t.Errorf("Round update failed: %s", err)
	}
	// Call ReceiveGatewayPingReport with bad auth
	report := &pb.GatewayPingReport{
		RoundId: roundId,
	}

	// Create host and auth
	params := connect.GetDefaultHostParams()
	params.AuthEnabled = false
	h, _ := connect.NewHost(instance.GetGateway(), testGatewayAddress, nil, params)
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          h,
	}

	err = ReceiveGatewayPingReport(report, auth, instance)
	if err != nil {
		t.Errorf("Error received in happy path: %v", err)
	}

}

// Happy path: handles a failed round due to gateway issues
func TestReceiveGatewayPingReport_FailedGateway(t *testing.T) {
	instance, _ := mockInstance(t, mockSharePhaseImpl)

	// Add the certs to our network instance
	cert, _ := utils.ReadFile(testkeys.GetNodeCertPath())
	_, err := instance.GetNetwork().AddHost(&id.Permissioning, "", cert, connect.GetDefaultHostParams())
	if err != nil {
		t.Logf("Failed to create host, %v", err)
		t.Fail()
	}

	// Add to consensus
	roundId := uint64(7)
	newRoundInfo := &pb.RoundInfo{
		ID: roundId,
	}

	// Mocking permissioning server signing message
	err = testutils.SignRoundInfo(newRoundInfo, t)
	if err != nil {
		t.Errorf("failed to sign round info: %v", err)
	}

	err = instance.GetConsensus().RoundUpdate(newRoundInfo)
	if err != nil {
		t.Errorf("Round update failed: %s", err)
	}

	failedGateway := id.NewIdFromString("FailedGateway", id.Gateway, t)

	// Call ReceiveGatewayPingReport with bad auth
	report := &pb.GatewayPingReport{
		RoundId:        roundId,
		FailedGateways: [][]byte{failedGateway.Bytes()},
	}

	// Create host and auth
	params := connect.GetDefaultHostParams()
	params.AuthEnabled = false
	h, _ := connect.NewHost(instance.GetGateway(), testGatewayAddress, nil, params)
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          h,
	}

	err = ReceiveGatewayPingReport(report, auth, instance)
	if err != nil {
		t.Errorf("Error received in happy path: %v", err)
	}

	// A failed gateway should result in a call to ReportRoundFailure,
	// updating the state machine
	if instance.GetStateMachine().Get() != current.ERROR {
		t.Errorf("State did not update to failure after "+
			"a failed gateway was received. "+
			"\n\tExpected state: %v"+
			"\n\tReceived state: %v", current.ERROR, instance.GetStateMachine().Get())
	}

}

// Test that nothing is done when a round was completed
// prior to reception of pingReport
func TestReceiveGatewayPingReport_RoundComplete(t *testing.T) {
	instance, _ := mockInstance(t, mockSharePhaseImpl)

	// Add the certs to our network instance
	cert, _ := utils.ReadFile(testkeys.GetNodeCertPath())
	_, err := instance.GetNetwork().AddHost(&id.Permissioning, "", cert, connect.GetDefaultHostParams())
	if err != nil {
		t.Logf("Failed to create host, %v", err)
		t.Fail()
	}

	// Add to consensus
	roundId := uint64(7)
	newRoundInfo := &pb.RoundInfo{
		ID:    roundId,
		State: uint32(states.COMPLETED),
	}

	// Mocking permissioning server signing message
	err = testutils.SignRoundInfo(newRoundInfo, t)
	if err != nil {
		t.Errorf("failed to sign round info: %v", err)
	}

	err = instance.GetConsensus().RoundUpdate(newRoundInfo)
	if err != nil {
		t.Errorf("Round update failed: %s", err)
	}

	failedGateway := id.NewIdFromString("FailedGateway", id.Gateway, t)

	// Call ReceiveGatewayPingReport with bad auth
	report := &pb.GatewayPingReport{
		RoundId:        roundId,
		FailedGateways: [][]byte{failedGateway.Bytes()},
	}

	// Create host and auth
	params := connect.GetDefaultHostParams()
	params.AuthEnabled = false
	h, _ := connect.NewHost(instance.GetGateway(), testGatewayAddress, nil, params)
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          h,
	}

	err = ReceiveGatewayPingReport(report, auth, instance)
	if err != nil {
		t.Errorf("Error received in happy path: %v", err)
	}

	// Receiving a report after a round has completed should
	// not update state
	if instance.GetStateMachine().Get() == current.ERROR {
		t.Errorf("State updated unexpectedly. Shoud not update " +
			"after round was already completed.")
	}

}

// Error path: Auth errors
func TestReceiveGatewayPingReport_AuthErrors(t *testing.T) {
	instance, _, _, _ := setupTests(t, current.REALTIME)

	// Create host and auth
	params := connect.GetDefaultHostParams()
	params.AuthEnabled = false
	h, _ := connect.NewHost(instance.GetGateway(), testGatewayAddress, nil, params)
	auth := &connect.Auth{
		IsAuthenticated: false,
		Sender:          h,
	}

	expectedError := connect.AuthError(auth.Sender.GetId()).Error()

	// Call ReceiveGatewayPingReport with bad auth
	report := &pb.GatewayPingReport{}
	err := ReceiveGatewayPingReport(report, auth, &instance)
	if err.Error() != expectedError {
		t.Errorf("Did not receive expected error!"+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", connect.AuthError(auth.Sender.GetId()), err)
	}

	// Report with a gateway that is not our own
	badId := id.NewIdFromString("not our gateway", id.Gateway, t)
	badHost, _ := connect.NewHost(badId, testGatewayAddress, nil, params)
	auth = &connect.Auth{
		IsAuthenticated: false,
		Sender:          badHost,
	}

	expectedError = connect.AuthError(auth.Sender.GetId()).Error()

	err = ReceiveGatewayPingReport(report, auth, &instance)
	if err.Error() != expectedError {
		t.Errorf("Did not receive expected error!"+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", connect.AuthError(auth.Sender.GetId()), err)
	}

}
