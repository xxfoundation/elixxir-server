////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package io registration.go handles the endpoints for registration

package io

import (
	"crypto"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/connect"
	hash2 "gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/crypto/nonce"
	"gitlab.com/elixxir/crypto/registration"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/server"
)

// Handles a client request for a nonce during the client registration process
func RequestNonce(instance *server.Instance, salt []byte, RSAPubKey string,
	DHPubKey, RSASignedByRegistration, DHSignedByClientRSA []byte,
	auth *connect.Auth) ([]byte, []byte, error) {

	// Verify the sender is the authenticated gateway for this node
	if !auth.IsAuthenticated ||
		auth.Sender.GetId() != instance.GetGateway().String() {
		return nil, nil, connect.AuthError(auth.Sender.GetId())
	}

	grp := instance.GetGroup()
	sha := crypto.SHA256

	if !instance.IsRegistrationAuthenticated() {
		regPubKey := instance.GetRegServerPubKey()
		h := sha.New()
		h.Write([]byte(RSAPubKey))
		data := h.Sum(nil)

		err := rsa.Verify(regPubKey, sha, data, RSASignedByRegistration, nil)
		if err != nil {
			// Invalid signed Client public key, return an error
			return []byte{}, []byte{},
				errors.Errorf("verification of public key signature "+
					"from registration failed: %+v", err)
		}
	}

	// Assemble Client public key
	userPublicKey, err := rsa.LoadPublicKeyFromPem([]byte(RSAPubKey))

	if err != nil {
		return []byte{}, []byte{},
			errors.Errorf("Unable to decode client RSA Pub Key: %+v", err)
	}

	//Check that the Client DH public key is signed correctly
	h := sha.New()
	h.Write(DHPubKey)
	data := h.Sum(nil)

	err = rsa.Verify(userPublicKey, sha, data, DHSignedByClientRSA, nil)

	if err != nil {
		return []byte{}, []byte{},
			errors.Errorf("Client signature on DH key could not be verified: %+v", err)
	}

	// Generate UserID
	userId := registration.GenUserID(userPublicKey, salt)

	// Generate a nonce with a timestamp
	userNonce, err := nonce.NewNonce(nonce.RegistrationTTL)

	if err != nil {
		return []byte{}, []byte{}, err
	}

	//Generate an ephemeral DH key pair
	DHPriv := grp.RandomCoprime(grp.NewInt(1))
	DHPub := grp.ExpG(DHPriv, grp.NewInt(1))
	clientDHPub := grp.NewIntFromBytes(DHPubKey)

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
	return userNonce.Bytes(), DHPub.Bytes(), nil
}

// Handles nonce confirmation during the client registration process
func ConfirmRegistration(instance *server.Instance, UserID, Signature []byte,
	auth *connect.Auth) ([]byte, error) {

	// Verify the sender is the authenticated gateway for this node
	if !auth.IsAuthenticated ||
		auth.Sender.GetId() != instance.GetGateway().String() {
		return nil, connect.AuthError(auth.Sender.GetId())
	}

	// Obtain the user from the database
	user, err := instance.GetUserRegistry().GetUser(id.NewUserFromBytes(UserID))

	if err != nil {
		// Invalid nonce, return an error
		return make([]byte, 0),
			errors.Errorf("Unable to confirm registration, could not "+
				"find a user: %+v", err)
	}

	// Verify nonce has not expired
	if !user.Nonce.IsValid() {
		return make([]byte, 0),
			errors.Errorf("Unable to confirm registration, Nonce is expired")
	}

	// Verify signed nonce using Client public key
	sha := crypto.SHA256

	h := sha.New()
	h.Write(user.Nonce.Bytes())
	data := h.Sum(nil)

	err = rsa.Verify(user.RsaPublicKey, sha, data, Signature, nil)

	if err != nil {
		return make([]byte, 0),
			errors.Errorf("Unable to confirm registration, signature invalid")
	}

	//todo: re-enable this and use it to simplify registration

	/*// Use  Server private key to sign Client public key
	userPubKeyPEM := rsa.CreatePublicKeyPem(user.RsaPublicKey)
	h.Reset()
	h.Write(userPubKeyPEM)
	data = h.Sum(nil)
	sig, err := rsa.Sign(csprng.NewSystemRNG(), instance.GetPrivKey(), sha, data, nil)
	if err != nil {
		// Unable to sign public key, return an error
		jww.ERROR.Printf("Error signing client public key: %s", err)
		return make([]byte, 0),	errors.New("unable to sign client public key")
	}*/

	//update the user's state to registered
	user.IsRegistered = true
	instance.GetUserRegistry().UpsertUser(user)
	// Fixme: what is going on here?
	return make([]byte, 0), nil
}
