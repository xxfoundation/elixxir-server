////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// Package io registration.go handles the endpoints for registration

package io

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/crypto/registration"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/chacha"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/crypto/xx"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"google.golang.org/protobuf/proto"
)

// RequestClientKey handles a client request for a nonce during the
// client registration process.
func RequestClientKey(instance *internal.Instance,
	request *pb.SignedClientKeyRequest, auth *connect.Auth) (*pb.SignedKeyResponse, error) {
	jww.WARN.Printf("received client key req")
	// Verify the sender is the authenticated gateway for this node
	if !auth.IsAuthenticated || !auth.Sender.GetId().Cmp(instance.GetGateway()) {
		errMsg := connect.AuthError(auth.Sender.GetId())
		return &pb.SignedKeyResponse{Error: errMsg.Error()}, errMsg
	}

	// get the group, if it cant be found return an error because we are not
	// ready
	grp := instance.GetNetworkStatus().GetCmixGroup()
	if grp == nil {
		errMsg := errors.New(ndf.NO_NDF)
		return &pb.SignedKeyResponse{Error: errMsg.Error()}, errMsg
	}

	// Parse serialized data into messageNoSecretExistsError
	msg := &pb.ClientKeyRequest{}
	err := proto.Unmarshal(request.ClientKeyRequest, msg)
	if err != nil {
		errMsg := errors.Errorf("Couldn't parse client key request: %v", err)
		return &pb.SignedKeyResponse{Error: errMsg.Error()}, errMsg
	}

	// Construct hash
	opts := rsa.NewDefaultOptions()
	opts.Hash = hash.CMixHash
	h := opts.Hash.New()

	// Parse serialized transmission confirmation into message
	clientTransmissionConfirmation := &pb.ClientRegistrationConfirmation{}
	err = proto.Unmarshal(msg.ClientTransmissionConfirmation.
		ClientRegistrationConfirmation, clientTransmissionConfirmation)
	if err != nil {
		errMsg := errors.Errorf("Couldn't parse client registration confirmation: %v", err)
		return &pb.SignedKeyResponse{Error: errMsg.Error()}, errMsg
	}

	// Extract RSA pubkey
	clientRsaPub := clientTransmissionConfirmation.RSAPubKey

	// Retrieve client registrar public key
	regPubKey := instance.GetRegServerPubKey()

	// Verify the registration signedResponse provided by the user
	err = registration.VerifyWithTimestamp(regPubKey, msg.RegistrationTimestamp,
		clientRsaPub,
		msg.ClientTransmissionConfirmation.RegistrarSignature.
			GetSignature())
	if err != nil {
		// Invalid signed Client public key, return an error
		errMsg := errors.Errorf("verification of public key signature "+
			"from registration failed: %+v", err)

		return &pb.SignedKeyResponse{Error: errMsg.Error()}, errMsg
	}

	// Assemble Client public key into rsa.PublicKey
	userPublicKey, err := rsa.LoadPublicKeyFromPem([]byte(clientRsaPub))
	if err != nil {
		errMsg := errors.Errorf("Unable to decode client RSA Pub Key: %+v", err)
		return &pb.SignedKeyResponse{Error: errMsg.Error()}, errMsg
	}

	// Reconstruct hashed data for signedResponse verification
	h.Reset()
	h.Write(request.ClientKeyRequest)
	data := h.Sum(nil)

	// Verify the signedResponse
	err = rsa.Verify(userPublicKey, hash.CMixHash, data,
		request.GetClientKeyRequestSignature().GetSignature(), opts)
	if err != nil {
		errMsg := errors.Errorf("Client signedResponse on client public key "+
			"could not be verified: %+v", err)

		return &pb.SignedKeyResponse{Error: errMsg.Error()}, errMsg
	}

	//Generate an ephemeral DH key pair
	DHPriv := grp.RandomCoprime(grp.NewInt(1))
	DHPub := grp.ExpG(DHPriv, grp.NewInt(1))

	if !csprng.InGroup(msg.GetClientDHPubKey(), grp.GetPBytes()) {
		return nil, errors.Errorf("Cannot process client request, "+
			"DH pub key is out of group: %v", msg.GetClientDHPubKey())
	}

	clientDHPub := grp.NewIntFromBytes(msg.GetClientDHPubKey())

	// Generate user CMIX baseKey
	h.Reset()
	sessionKey := registration.GenerateBaseKey(grp, clientDHPub, DHPriv, h)

	// Generate UserID
	userId, err := xx.NewID(userPublicKey, msg.GetSalt(), id.User)
	if err != nil {
		errMsg := errors.Errorf("Failed to generate new ID: %+v", err)

		return &pb.SignedKeyResponse{Error: errMsg.Error()}, errMsg
	}

	// Retrieve node secret
	// TODO: replace hardcoding of keyID once multiple rotating node secrets
	//  is supported
	nodeSecret, err := instance.GetSecretManager().GetSecret(0)
	if err != nil {
		return nil, err
	}

	// Construct client key
	h.Reset()
	h.Write(userId.Bytes())
	h.Write(nodeSecret.Bytes())
	clientKey := h.Sum(nil)

	// Construct client gateway key
	h.Reset()
	h.Write(clientKey)
	clientGatewayKey := h.Sum(nil)

	// Encrypt the client key using the session key
	encryptedClientKey, err := chacha.Encrypt(sessionKey.Bytes(), clientKey,
		instance.GetRngStreamGen().GetStream())
	if err != nil {
		errMsg := errors.Errorf("Unable to encrypt key: %v", err)
		return &pb.SignedKeyResponse{Error: errMsg.Error()}, errMsg
	}

	// Construct HMAC
	h.Reset()
	encryptedClientKeyHMAC := registration.CreateClientHMAC(sessionKey.Bytes(),
		encryptedClientKey, opts.Hash.New)

	jww.TRACE.Printf("[ClientKeyHMAC] Session Key Bytes: %+v", sessionKey.Bytes())
	jww.TRACE.Printf("[ClientKeyHMAC] EncryptedClientKey: %+v", encryptedClientKey)
	jww.TRACE.Printf("[ClientKeyHMAC] EncryptedClientKeyHMAC: %+v", encryptedClientKeyHMAC)

	// Construct response
	resp := &pb.ClientKeyResponse{
		EncryptedClientKey:     encryptedClientKey,
		EncryptedClientKeyHMAC: encryptedClientKeyHMAC,
		NodeDHPubKey:           DHPub.Bytes(),
		// Fixme: the values below are future features for
		//  expiring nodeSecret values.
		//  KeyID identifies the active key identities that can still be used
		//  ValidUntil denotes the time at which the key id retrieved becomes
		//  invalid to the server
		//  This is how we provide a form of forward secrecy in the case where
		//  NodeSecrets are leaked
		KeyID:      nil,
		ValidUntil: 0,
	}

	// Serialize response
	serializedResponse, err := proto.Marshal(resp)
	if err != nil {
		errMsg := errors.Errorf("Could not serialize response: %v", err)
		return &pb.SignedKeyResponse{Error: errMsg.Error()}, errMsg
	}

	// Return signed key response to Client with empty error field
	return &pb.SignedKeyResponse{
		KeyResponse:      serializedResponse,
		ClientGatewayKey: clientGatewayKey,
		Error:            "",
	}, nil
}
