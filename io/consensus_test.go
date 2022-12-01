////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"bytes"
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/testkeys"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/server/testUtil"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/signature"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	jww.SetStdoutThreshold(jww.LevelTrace)
	connect.TestingOnlyDisableTLS = true
	os.Exit(m.Run())
}

func TestGetNdf(t *testing.T) {
	instance, _, _ := createMockInstance(t, 0, current.REALTIME)

	certPath := testkeys.GetGatewayCertPath()
	cert := testkeys.LoadFromPath(certPath)
	params := connect.GetDefaultHostParams()
	params.AuthEnabled = false
	_, err := instance.GetNetwork().AddHost(&id.Permissioning, "", cert, params)
	if err != nil {
		t.Fatalf("Failed to create host, %v", err)
	}

	key := testkeys.LoadFromPath(testkeys.GetGatewayKeyPath())
	privateKey, err := rsa.LoadPrivateKeyFromPem(key)
	if err != nil {
		t.Errorf("Failed to load private key: %+v", err)
	}

	ndfMsg := &pb.NDF{
		Ndf: testUtil.ExampleNDF,
	}
	err = signature.SignRsa(ndfMsg, privateKey)
	if err != nil {
		t.Errorf("Failed to RSA sign NDF: %+v", err)
	}

	err = instance.GetNetworkStatus().UpdateFullNdf(ndfMsg)
	if err != nil {
		t.Errorf("Failed to update NDF: %+v", err)
	}

	receivedNdf, err := GetNdf(instance)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !bytes.Equal(receivedNdf, testUtil.ExampleNDF) {
		t.Errorf("Did not get expected result!"+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", testUtil.ExampleNDF, receivedNdf)
	}
}
