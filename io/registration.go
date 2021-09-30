///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Package io registration.go handles the endpoints for registration

package io

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cmix"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/crypto/registration"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/storage"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/crypto/chacha"
	"gitlab.com/xx_network/crypto/nonce"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/crypto/xx"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"google.golang.org/protobuf/proto"
)

// RequestRequestClientKey handles a client request for a nonce during the client registration process
func RequestRequestClientKey(instance *internal.Instance,
	request *pb.SignedClientKeyRequest, auth *connect.Auth) (*pb.SignedKeyResponse, error) {

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
		msg.GetClientTransmissionConfirmation().GetRegistrarSignature().
			GetSignature())
	if err != nil {
		// Invalid signed Client public key, return an error
		errMsg := errors.Errorf("verification of public key signedResponse "+
			"from registration failed: %+v", err)

		return &pb.SignedKeyResponse{Error: errMsg.Error()}, errMsg
	}

	// Assemble Client public key
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
		errMsg := errors.Errorf("Client signedResponse on DH key could not be verified: %+v", err)

		return &pb.SignedKeyResponse{Error: errMsg.Error()}, errMsg
	}

	//Generate an ephemeral DH key pair
	DHPriv := grp.RandomCoprime(grp.NewInt(1))
	DHPub := grp.ExpG(DHPriv, grp.NewInt(1))
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
	//TODO: replace hardcoding of keyID once multiple rotating node secrets
	// is supported
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
		encryptedClientKey, h)

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

	// Hash serialized response
	h.Reset()
	h.Write(serializedResponse)
	hashedResponse := h.Sum(nil)

	// Sign the response
	signedResponse, err := rsa.Sign(instance.GetRngStreamGen().GetStream(), instance.GetPrivKey(),
		opts.Hash, hashedResponse, opts)
	if err != nil {
		errMsg := errors.Errorf("Could not sign key response: %v", err)
		return &pb.SignedKeyResponse{Error: errMsg.Error()}, errMsg
	}

	// Return signed key response to Client with empty error field
	return &pb.SignedKeyResponse{
		KeyResponse:             serializedResponse,
		KeyResponseSignedByNode: &messages.RSASignature{Signature: signedResponse},
		ClientGatewayKey:        clientGatewayKey,
		Error:                   "",
	}, nil
}

// ---------------------- Start of deprecated fields ----------- //

// RequestNonce handles a client request for a nonce during the client registration process
// TODO: Remove comm once RequestClientKey is properly tested
func RequestNonce(instance *internal.Instance,
	request *pb.NonceRequest, auth *connect.Auth) (*pb.Nonce, error) {

	jww.WARN.Printf("DEPRECATED: Client is registering using soon to be deprecated code path")

	// Verify the sender is the authenticated gateway for this node
	if !auth.IsAuthenticated || !auth.Sender.GetId().Cmp(instance.GetGateway()) {
		return &pb.Nonce{}, connect.AuthError(auth.Sender.GetId())
	}

	// get the group, if it cant be found return an error because we are not
	// ready
	grp := instance.GetNetworkStatus().GetCmixGroup()
	if grp == nil {
		return &pb.Nonce{}, errors.New(ndf.NO_NDF)
	}

	regPubKey := instance.GetRegServerPubKey()

	// Verify the registration signature provided by the user
	err := registration.VerifyWithTimestamp(regPubKey, request.TimeStamp,
		request.GetClientRSAPubKey(), request.GetClientSignedByServer().Signature)
	if err != nil {
		// Invalid signed Client public key, return an error
		return &pb.Nonce{},
			errors.Errorf("verification of public key signature "+
				"from registration failed: %+v", err)
	}

	// Assemble Client public key
	userPublicKey, err := rsa.LoadPublicKeyFromPem([]byte(request.GetClientRSAPubKey()))
	if err != nil {
		return &pb.Nonce{},
			errors.Errorf("Unable to decode client RSA Pub Key: %+v", err)
	}

	//Check that the Client DH public key is signed correctly
	h, _ := hash.NewCMixHash()
	h.Write(request.GetClientDHPubKey())
	data := h.Sum(nil)
	err = rsa.Verify(userPublicKey, hash.CMixHash, data,
		request.GetRequestSignature().Signature, nil)
	if err != nil {
		return &pb.Nonce{},
			errors.Errorf("Client signature on DH key could not be verified: %+v", err)
	}

	// Generate UserID
	userId, err := xx.NewID(userPublicKey, request.GetSalt(), id.User)
	if err != nil {
		return &pb.Nonce{},
			errors.Errorf("Failed to generate new ID: %+v", err)
	}

	// Generate a nonce with a timestamp
	userNonce, err := nonce.NewNonce(nonce.RegistrationTTL)
	if err != nil {
		return &pb.Nonce{}, err
	}

	//Generate an ephemeral DH key pair
	DHPriv := grp.RandomCoprime(grp.NewInt(1))
	DHPub := grp.ExpG(DHPriv, grp.NewInt(1))
	clientDHPub := grp.NewIntFromBytes(request.GetClientDHPubKey())

	// Generate user CMIX baseKey
	b, _ := hash.NewCMixHash()
	baseKey := registration.GenerateBaseKey(grp, clientDHPub, DHPriv, b)

	// Store user information in the database
	newClient := &storage.Client{
		Id:             userId.Bytes(),
		DhKey:          baseKey.Bytes(),
		PublicKey:      rsa.CreatePublicKeyPem(userPublicKey),
		Nonce:          userNonce.Bytes(),
		NonceTimestamp: userNonce.GenTime,
		IsRegistered:   false,
	}
	err = instance.GetStorage().UpsertClient(newClient)
	if err != nil {
		return nil, err
	}

	// Return nonce to Client with empty error field
	return &pb.Nonce{
		Nonce:    userNonce.Bytes(),
		DHPubKey: DHPub.Bytes(),
	}, nil
}

// ConfirmRegistration handles nonce confirmation during the client registration process
// TODO: Remove comm once RequestClientKey is properly tested
func ConfirmRegistration(instance *internal.Instance, confirmation *pb.RequestRegistrationConfirmation,
	auth *connect.Auth) (*pb.RegistrationConfirmation, error) {

	jww.WARN.Printf("DEPRECATED: Client is registering using soon to be deprecated code path")

	// Verify the sender is the authenticated gateway for this node
	if !auth.IsAuthenticated || !auth.Sender.GetId().Cmp(instance.GetGateway()) {
		return &pb.RegistrationConfirmation{}, connect.AuthError(auth.Sender.GetId())
	}

	UserID, err := id.Unmarshal(confirmation.GetUserID())
	if err != nil {
		return &pb.RegistrationConfirmation{}, errors.Errorf("Unable to unmarshal client ID: %+v", err)
	}

	Signature := confirmation.NonceSignedByClient.Signature

	// Obtain the client from the database
	client, err := instance.GetStorage().GetClient(UserID)
	if err != nil {
		// Invalid nonce, return an error
		return &pb.RegistrationConfirmation{},
			errors.Errorf("Unable to confirm registration, could not "+
				"find a client: %+v", err)
	}

	// Verify nonce has not expired
	clientNonce := client.GetNonce()
	if !clientNonce.IsValid() {
		return &pb.RegistrationConfirmation{},
			errors.Errorf("Unable to confirm registration, Nonce is expired")
	}

	// Verify signed nonce and our ID using Client public key
	h, _ := hash.NewCMixHash()
	h.Write(clientNonce.Bytes())
	h.Write(instance.GetID().Bytes())
	data := h.Sum(nil)

	pubKey, err := client.GetPublicKey()
	if err != nil {
		return nil, err
	}

	err = rsa.Verify(pubKey, hash.CMixHash, data, Signature, nil)
	if err != nil {
		return &pb.RegistrationConfirmation{},
			errors.WithMessagef(err, "Unable to confirm registration with %s, signature invalid: %+v", client.Id, err)
	}

	//todo: re-enable this and use it to simplify registration

	/*// Use  Server private key to sign Client public key
	userPubKeyPEM := rsa.CreatePublicKeyPem(client.RsaPublicKey)
	h.Reset()
	h.Write(userPubKeyPEM)
	data = h.Sum(nil)
	sig, err := rsa.Sign(csprng.NewSystemRNG(), instance.GetPrivKey(), hash2.CMixHash, data, nil)
	if err != nil {
		// Unable to sign public key, return an error
		jww.ERROR.Printf("Error signing client public key: %s", err)
		return make([]byte, 0),	errors.New("unable to sign client public key")
	}*/

	// Hash the basekey
	hashedData := cmix.GenerateClientGatewayKey(client.GetDhKey(instance.GetNetworkStatus().GetCmixGroup()))
	//update the client's state to registered
	client.IsRegistered = true
	err = instance.GetStorage().UpsertClient(client)
	if err != nil {
		return nil, err
	}
	// Fixme: what is going on here? RSA signature has been blank?
	response := &pb.RegistrationConfirmation{
		ClientSignedByServer: &messages.RSASignature{
			Signature: make([]byte, 0),
		},
		ClientGatewayKey: hashedData,
	}
	return response, nil
}

// ---------------------- End of deprecated fields ----------- //
