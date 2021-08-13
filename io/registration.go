///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Package io registration.go handles the endpoints for registration

package io

import (
	"fmt"
	"github.com/pkg/errors"
	pb "git.xx.network/elixxir/comms/mixmessages"
	"git.xx.network/elixxir/crypto/cmix"
	"git.xx.network/elixxir/crypto/hash"
	"git.xx.network/elixxir/crypto/registration"
	"git.xx.network/elixxir/server/internal"
	"git.xx.network/elixxir/server/storage"
	"git.xx.network/xx_network/comms/connect"
	"git.xx.network/xx_network/comms/messages"
	"git.xx.network/xx_network/crypto/nonce"
	"git.xx.network/xx_network/crypto/signature/rsa"
	"git.xx.network/xx_network/crypto/xx"
	"git.xx.network/xx_network/primitives/id"
	"git.xx.network/xx_network/primitives/ndf"
)

// RequestNonce handles a client request for a nonce during the client registration process
func RequestNonce(instance *internal.Instance,
	request *pb.NonceRequest, auth *connect.Auth) (*pb.Nonce, error) {

	fmt.Printf("Sender ID:  %#v\n", auth.Sender.GetId())
	fmt.Printf("Gateway ID: %#v\n", instance.GetGateway())

	// Verify the sender is the authenticated gateway for this node
	if !auth.IsAuthenticated || !auth.Sender.GetId().Cmp(instance.GetGateway()) {
		return &pb.Nonce{}, connect.AuthError(auth.Sender.GetId())
	}

	// get the group, if it cant be found return an error because we are not
	// ready
	grp := instance.GetConsensus().GetCmixGroup()
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
func ConfirmRegistration(instance *internal.Instance, confirmation *pb.RequestRegistrationConfirmation,
	auth *connect.Auth) (*pb.RegistrationConfirmation, error) {

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
	hashedData := cmix.GenerateClientGatewayKey(client.GetDhKey(instance.GetConsensus().GetCmixGroup()))
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
