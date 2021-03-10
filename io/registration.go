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
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cmix"
	hash2 "gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/crypto/registration"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/crypto/nonce"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/crypto/xx"
	"gitlab.com/xx_network/primitives/id"
)

// Handles a client request for a nonce during the client registration process
func RequestNonce(instance *internal.Instance,
	request *pb.NonceRequest, auth *connect.Auth) (*pb.Nonce, error) {

	fmt.Printf("Sender ID:  %#v\n", auth.Sender.GetId())
	fmt.Printf("Gateway ID: %#v\n", instance.GetGateway())

	// Verify the sender is the authenticated gateway for this node
	if !auth.IsAuthenticated || !auth.Sender.GetId().Cmp(instance.GetGateway()) {
		return &pb.Nonce{}, connect.AuthError(auth.Sender.GetId())
	}

	grp := instance.GetConsensus().GetCmixGroup()

	regPubKey := instance.GetRegServerPubKey()
	h, _ := hash2.NewCMixHash()
	h.Write([]byte(request.GetClientRSAPubKey()))
	data := h.Sum(nil)

	err := rsa.Verify(regPubKey, hash2.CMixHash, data,
		request.GetClientSignedByServer().Signature, nil)
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
	h, _ = hash2.NewCMixHash()
	h.Write(request.GetClientDHPubKey())
	data = h.Sum(nil)

	err = rsa.Verify(userPublicKey, hash2.CMixHash, data,
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
	b, _ := hash2.NewCMixHash()
	baseKey := registration.GenerateBaseKey(grp, clientDHPub, DHPriv, b)

	// Store user information in the database
	newUser := instance.GetUserRegistry().NewUser(grp)
	newUser.Nonce = userNonce
	newUser.ID = userId
	newUser.RsaPublicKey = userPublicKey
	newUser.BaseKey = baseKey
	newUser.IsRegistered = false
	instance.GetUserRegistry().UpsertUser(newUser)

	// Return nonce to Client with empty error field
	return &pb.Nonce{
		Nonce:    userNonce.Bytes(),
		DHPubKey: DHPub.Bytes(),
	}, nil
}

// Handles nonce confirmation during the client registration process
func ConfirmRegistration(instance *internal.Instance, confirmation *pb.RequestRegistrationConfirmation,
	auth *connect.Auth) (*pb.RegistrationConfirmation, error) {

	// Verify the sender is the authenticated gateway for this node
	if !auth.IsAuthenticated || !auth.Sender.GetId().Cmp(instance.GetGateway()) {
		return &pb.RegistrationConfirmation{}, connect.AuthError(auth.Sender.GetId())
	}

	UserID, err := id.Unmarshal(confirmation.GetUserID())
	if err != nil {
		return &pb.RegistrationConfirmation{}, errors.Errorf("Unable to unmarshal user ID: %+v", err)
	}

	Signature := confirmation.NonceSignedByClient.Signature

	// Obtain the user from the database
	user, err := instance.GetUserRegistry().GetUser(UserID, instance.GetConsensus().GetCmixGroup())

	if err != nil {
		// Invalid nonce, return an error
		return &pb.RegistrationConfirmation{},
			errors.Errorf("Unable to confirm registration, could not "+
				"find a user: %+v", err)
	}

	// Verify nonce has not expired
	if !user.Nonce.IsValid() {
		return &pb.RegistrationConfirmation{},
			errors.Errorf("Unable to confirm registration, Nonce is expired")
	}

	// Verify signed nonce and our ID using Client public key
	h, _ := hash2.NewCMixHash()
	h.Write(user.Nonce.Bytes())
	h.Write(instance.GetID().Bytes())
	data := h.Sum(nil)
	// todo: remove this print
	jww.INFO.Printf("ConfirmRegistration hashedData from user [%v]: %v", user.ID, data)
	err = rsa.Verify(user.RsaPublicKey, hash2.CMixHash, data, Signature, nil)

	if err != nil {
		return &pb.RegistrationConfirmation{},
			errors.WithMessagef(err, "Unable to confirm registration with %s, signature invalid: %s", user.ID)
	}

	//todo: re-enable this and use it to simplify registration

	/*// Use  Server private key to sign Client public key
	userPubKeyPEM := rsa.CreatePublicKeyPem(user.RsaPublicKey)
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
	hashedData := cmix.GenerateClientGatewayKey(user.BaseKey)
	user.BaseKey.Bytes()
	//update the user's state to registered
	user.IsRegistered = true
	instance.GetUserRegistry().UpsertUser(user)
	// Fixme: what is going on here? RSA signature has been blank?
	response := &pb.RegistrationConfirmation{
		ClientSignedByServer: &messages.RSASignature{
			Signature: make([]byte, 0),
		},
		ClientGatewayKey: hashedData,
	}
	return response, nil
}
