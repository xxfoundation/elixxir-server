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

	if ourNewReport == nil {
		t.Errorf("New Client report should not be nil: %+v", ourNewReport)
	}

	rndId := id.Round(0)
	ourNewReport.ErrorTracker[rndId] = make(chan *pb.ClientError, 8)
	if len(ourNewReport.ErrorTracker) != 1 {
		t.Errorf("Client report expected to be of length 1! Length is: %+v", len(ourNewReport.ErrorTracker))
	}

	ce := &pb.ClientError{}

	// Test
	ourNewReport.ErrorTracker[rndId] <- ce

	if len(ourNewReport.ErrorTracker[rndId]) != 1 {
		t.Errorf("Client report expected to be of length 1! "+
			"Length is: %+v", len(ourNewReport.ErrorTracker[rndId]))
	}

}

// Happy path
func TestClientReport_Send(t *testing.T) {
	ourNewReport := NewClientFailureReport()
	rndId := id.Round(0)

	ourNewReport.ErrorTracker[rndId] = make(chan *pb.ClientError, 8)

	clientErr := &pb.ClientError{}
	err := ourNewReport.Send(rndId, clientErr)
	if len(ourNewReport.ErrorTracker) != 1 {
		t.Errorf("Error tracker should have length of 1 after a report! "+
			"Length is: %+v", len(ourNewReport.ErrorTracker))
	}

	if err != nil {
		t.Errorf("Unexpcted error: %v", err)
	}

	err = ourNewReport.Send(rndId, clientErr)
	if err != nil {
		t.Errorf("Should be able to send when reporter is empty: %+v."+
			"\nLength of reporter: %+v", err, len(ourNewReport.ErrorTracker))
	}

	if len(ourNewReport.ErrorTracker[rndId]) != 2 {
		t.Errorf("Error tracker should be two after a send! "+
			"Length is: %+v", len(ourNewReport.ErrorTracker[rndId]))

	}

}

//
//// Happy path
func TestClientReport_Receive_Receive(t *testing.T) {
	ourNewReport := NewClientFailureReport()
	testId := id.NewIdFromBytes([]byte("test"), t)
	testErr := "I failed due to an invalid KMAC"
	ce := &pb.ClientError{
		ClientId: testId.Bytes(),
		Error:    testErr,
	}

	rndId := id.Round(0)
	ourNewReport.ErrorTracker[rndId] = make(chan *pb.ClientError, 8)

	// Send to queue
	err := ourNewReport.Send(rndId, ce)
	if err != nil {
		t.Errorf("Expected happy path, received error when sending! Err: %+v", err)
	}

	receivedClientErrs, err := ourNewReport.Receive(rndId)
	if err != nil {
		t.Errorf("Expected happy path, received error when receiving! Err: %+v", err)
	}

	if len(receivedClientErrs) != 1 {
		t.Logf("Received unexpected round id")
		t.Fail()
	}

	if !reflect.DeepEqual(receivedClientErrs[0], ce) {
		t.Errorf("Client error received from channel does not match input from channel."+
			"\n\tReceived: %v"+
			"\n\tExpected: %v", receivedClientErrs[0], ce)
	}

}

// Error path: Attempt to receive from an empty queue
func TestClientReport_Receive_Error(t *testing.T) {
	ourNewReport := NewClientFailureReport()
	rndID := id.Round(0)
	_, err := ourNewReport.Receive(rndID)

	if err != nil {
		return
	}

	t.Errorf("Expected error path, should not be able to receive from an empty queue!")
}
