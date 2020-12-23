///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////
package io

import (
	"errors"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"testing"
)

type permComms struct{}

func (c *permComms) GetHost(hostId *id.ID) (*connect.Host, bool) {
	h := connect.Host{}
	return &h, true
}

func (c *permComms) SendCheckConnectivityMessage(host *connect.Host, message *pb.Address) (*pb.ConnectivityResponse, error) {
	return &pb.ConnectivityResponse{
		CallerAddr:      "192.168.1.1",
		CallerAvailable: true,
		OtherAvailable:  true,
	}, nil
}

// This test tests that the default IP of "0.0.0.0" is not returned,
// instead the detected one
func TestCheckPermConn(t *testing.T) {
	pc := permComms{}

	// Run our function
	addr, err := TransmitSendCheckConnectivity("0.0.0.0", 6550, &pc)
	if err != nil {
		t.Fatal(err)
	}

	// Check that it returned our IP
	if addr != "192.168.1.1" {
		t.Fatalf("CheckPermConn returned IP %v instead of 0.0.0.0", addr)
	}
}

// This test tests that the function returns our config IP, instead of the reported one
func TestCheckPermConn_SetConfigIP(t *testing.T) {
	pc := permComms{}

	// Run our function
	addr, err := TransmitSendCheckConnectivity("192.168.15.6", 6550, &pc)
	if err != nil {
		t.Fatal(err)
	}

	// Check that it returned our IP
	if addr != "192.168.15.6" {
		t.Fatalf("CheckPermConn returned IP %v instead of 0.0.0.0", addr)
	}
}

// ----------------- BAD PERM HOST TEST -----------------
// This test tests the error if it can't see the Permissioning server as a host
type permCommsBadPermHost struct{}

func (c *permCommsBadPermHost) GetHost(hostId *id.ID) (*connect.Host, bool) {
	return nil, false
}

func (c *permCommsBadPermHost) SendCheckConnectivityMessage(host *connect.Host, message *pb.Address) (*pb.ConnectivityResponse, error) {
	return nil, errors.New("i don't know how you even got here")
}

func TestCheckPermConn_NoPermHost(t *testing.T) {
	pc := permCommsBadPermHost{}
	_, err := TransmitSendCheckConnectivity("192.168.1.1", 6550, &pc)
	if err == nil {
		t.Errorf("CheckPermConn did not return an error")
	}
	if err.Error() != "CheckPermConn could not find permissioning host" {
		t.Errorf("CheckPermConn returned invalid error %v", err)
	}
}

// ----------------- BAD COMMS CHECK CONN TEST -----------------
// This test tests the error if it can't see the Permissioning server as a host
type permCommsBadCheckConn struct{}

func (c *permCommsBadCheckConn) GetHost(hostId *id.ID) (*connect.Host, bool) {
	h := connect.Host{}
	return &h, true
}

func (c *permCommsBadCheckConn) SendCheckConnectivityMessage(host *connect.Host, message *pb.Address) (*pb.ConnectivityResponse, error) {
	return nil, errors.New("i just don't know what went wrong")
}

func TestCheckPermConn_BadCommsCheckConn(t *testing.T) {
	pc := permCommsBadCheckConn{}
	_, err := TransmitSendCheckConnectivity("192.168.1.1", 6550, &pc)
	if err == nil {
		t.Errorf("CheckPermConn did not return an error")
	}
	if err.Error() != "i just don't know what went wrong" {
		t.Errorf("CheckPermConn returned invalid error %v", err)
	}
}
