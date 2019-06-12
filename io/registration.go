////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package io registration.go handles the endpoints for registration

package io

import (
	"bytes"
	"crypto/rand"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	hash2 "gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/crypto/nonce"
	"gitlab.com/elixxir/crypto/registration"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/server/server"
)

func RequestNonce(instance *server.Instance,
	salt, Y, P, Q, G, hash, R, S []byte) ([]byte, error) {

	grp := instance.GetGroup()
	privKey := instance.GetPrivKey()

	if !instance.GetSkipReg() {
		// Verify signed public key using hardcoded RegistrationServer public key
		valid := instance.GetRegPubKey().Verify(hash, signature.DSASignature{
			R: large.NewIntFromBytes(R),
			S: large.NewIntFromBytes(S),
		})

		// Concatenate Client public key byte slices
		data := make([]byte, 0)
		data = append(data, Y...)
		data = append(data, P...)
		data = append(data, Q...)
		data = append(data, G...)

		// Ensure that the data in the hash is identical to the Client public key
		if !valid || !bytes.Equal(data, hash) {
			// Invalid signed Client public key, return an error
			jww.ERROR.Printf("Unable to verify signed public key!")
			return make([]byte, 0), errors.Errorf("signed public key is invalid")
		}
	}

	// Assemble Client public key
	userPublicKey := signature.ReconstructPublicKey(
		signature.CustomDSAParams(
			large.NewIntFromBytes(P),
			large.NewIntFromBytes(Q),
			large.NewIntFromBytes(G)),
		large.NewIntFromBytes(Y))

	// Generate UserID
	userId := registration.GenUserID(userPublicKey, salt)

	// Generate a nonce with a timestamp
	userNonce := nonce.NewNonce(nonce.RegistrationTTL)

	// Generate user CMIX baseKey
	b, _ := hash2.NewCMixHash()
	baseKey := registration.GenerateBaseKey(grp, userPublicKey, privKey, b)

	// Store user information in the database
	newUser := instance.GetUserRegistry().NewUser(grp)
	newUser.Nonce = userNonce
	newUser.ID = userId
	newUser.PublicKey = userPublicKey
	newUser.BaseKey = baseKey
	instance.GetUserRegistry().UpsertUser(newUser)

	// Return nonce to Client with empty error field
	return userNonce.Bytes(), nil
}

func ConfirmRegistration(instance *server.Instance,
	hash, R, S []byte) ([]byte, []byte, []byte, []byte, []byte,
	[]byte, []byte, error) {

	// Obtain the user from the database
	n := nonce.Nonce{}
	copy(n.Value[:], hash)
	user, err := instance.GetUserRegistry().GetUserByNonce(n)

	if err != nil {
		// Invalid nonce, return an error
		jww.ERROR.Printf("Unable to find nonce: %x", n.Bytes())
		return make([]byte, 0), make([]byte, 0), make([]byte, 0),
			make([]byte, 0), make([]byte, 0), make([]byte, 0), make([]byte, 0),
			errors.Errorf("nonce does not exist")
	}

	// Verify nonce has not expired
	if !user.Nonce.IsValid() {
		jww.ERROR.Printf("Nonce is expired: %x", n.Bytes())
		return make([]byte, 0), make([]byte, 0), make([]byte, 0),
			make([]byte, 0), make([]byte, 0), make([]byte, 0), make([]byte, 0),
			errors.Errorf("nonce is expired")
	}

	// Verify signed nonce using Client public key
	valid := user.PublicKey.Verify(hash, signature.DSASignature{
		R: large.NewIntFromBytes(R),
		S: large.NewIntFromBytes(S),
	})

	if !valid {
		// Invalid signed nonce, return an error
		jww.ERROR.Printf("Unable to verify nonce: %x", n.Bytes())
		return make([]byte, 0), make([]byte, 0), make([]byte, 0),
			make([]byte, 0), make([]byte, 0), make([]byte, 0), make([]byte, 0),
			errors.Errorf("signed nonce is invalid")
	}

	// Concatenate Client public key byte slices
	data := make([]byte, 0)
	params := user.PublicKey.GetParams()
	data = append(data, user.PublicKey.GetKey().Bytes()...)
	data = append(data, params.GetP().Bytes()...)
	data = append(data, params.GetQ().Bytes()...)
	data = append(data, params.GetG().Bytes()...)

	// Use hardcoded Server private key to sign Client public key
	sig, err := instance.GetPrivKey().Sign(data, rand.Reader)
	if err != nil {
		// Unable to sign public key, return an error
		jww.ERROR.Printf("Error signing client public key: %s", err)
		return make([]byte, 0), make([]byte, 0), make([]byte, 0),
			make([]byte, 0), make([]byte, 0), make([]byte, 0), make([]byte, 0),
			errors.New("unable to sign client public key")
	}

	grp := instance.GetGroup()
	// Return the signed Client public key to Client with empty error field
	return data, sig.R.Bytes(), sig.S.Bytes(), instance.GetPubKey().GetKey().Bytes(),
		grp.GetPBytes(), grp.GetQ().Bytes(), grp.GetG().Bytes(), nil
}
