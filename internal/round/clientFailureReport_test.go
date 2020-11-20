///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////
package round

import (
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"testing"
)

// Smoke test of new clientReport
func TestNewClientReport(t *testing.T) {
	ourNewReport := NewClientFailureReport()

	if len(ourNewReport.userErrorChannel) != 0 {
		t.Errorf("New Client report expected to be of length 0! Length is: %+v", len(ourNewReport.userErrorChannel))
	}

	clientErrs := &pb.ClientErrors{}

	// Test
	ourNewReport.userErrorChannel <- clientErrs

	if len(ourNewReport.userErrorChannel) != 1 {
		t.Errorf("Client report expected to be of length 1! Length is: %+v", len(ourNewReport.userErrorChannel))
	}

}

// Happy path
func TestClientReport_Send(t *testing.T) {
	ourNewReport := NewClientFailureReport()

	if len(ourNewReport.userErrorChannel) != 0 {
		t.Errorf("New Client report expected to be of length 0! Length is: %+v", len(ourNewReport.userErrorChannel))
	}
	clientErrs := &pb.ClientErrors{}
	rndId := uint64(0)
	ourNewReport.Report(clientErrs, rndId)
	if len(ourNewReport.userErrorTracker) != 1 {
		t.Errorf("Error tracker should have length of 1 after a report! "+
			"Length is: %+v", len(ourNewReport.userErrorTracker))

	}
	err := ourNewReport.Send(rndId)
	if err != nil {
		t.Errorf("Should be able to send when reporter is empty: %+v."+
			"\nLength of reporter: %+v", err, len(ourNewReport.userErrorChannel))
	}

	if len(ourNewReport.userErrorTracker) != 1 {
		t.Errorf("Error tracker should be empty after a send! "+
			"Length is: %+v", len(ourNewReport.userErrorChannel))

	}

}

// Happy path
func TestClientReport_Receive_Receive(t *testing.T) {
	ourNewReport := NewClientFailureReport()
	testId := id.NewIdFromBytes([]byte("test"), t)
	testErr := "I failed due to an invalid KMAC"
	ce := &pb.ClientError{
		ClientId: testId.Bytes(),
		Error:    testErr,
	}

	clientErrs := &pb.ClientErrors{ClientErrors: []*pb.ClientError{ce}}
	rndId := uint64(0)
	// Send to queue
	ourNewReport.Report(clientErrs, rndId)
	err := ourNewReport.Send(rndId)
	if err != nil {
		t.Errorf("Expected happy path, received error when sending! Err: %+v", err)
	}

	receivedClientErrs, err := ourNewReport.Receive()
	if err != nil {
		t.Errorf("Expected happy path, received error when receiving! Err: %+v", err)
	}

	if len(receivedClientErrs.ClientErrors) != 1 {
		t.Logf("Received unexpected round id")
		t.Fail()
	}

	if !reflect.DeepEqual(receivedClientErrs.ClientErrors[0], ce) {
		t.Errorf("Client error received from channel does not match input from channel."+
			"\n\tReceived: %v"+
			"\n\tExpected: %v", receivedClientErrs.ClientErrors[0], ce)
	}

}

// Error path: Attempt to receive from an empty queue
func TestClientReport_Receive_Error(t *testing.T) {
	ourNewReport := NewClientFailureReport()

	_, err := ourNewReport.Receive()

	if err != nil {
		return
	}

	t.Errorf("Expected error path, should not be able to receive from an empty queue!")
}
