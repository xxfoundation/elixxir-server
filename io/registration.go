////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles creating callbacks for registration hooks into comms library

package io

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/cyclic"
	hash2 "gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/crypto/nonce"
	"gitlab.com/elixxir/crypto/registration"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/server/globals"
)

// DSA Params
var dsaParams = signature.GetDefaultDSAParams()

// DSA Group
var dsaGrp = cyclic.NewGroup(dsaParams.GetP(), large.NewInt(2),
	dsaParams.GetG())

// Hardcoded DSA public key for Server
var publicKey = signature.ReconstructPublicKey(dsaParams,
	large.NewIntFromString("5c73ff5a2b9eea19d180334a60fa0299dbd4a4b724506cc0fd15f3ffd7b755c6"+
		"67cf0ce08a7a366b6bec808a6846d5aa1d047144048fa52522b083c7512dabf5"+
		"595204f7ca125618c27842b0658c05fe619a1400cc710b109f0f3b9dcf9faa35"+
		"4407a9c2b914584792cd60f0bc3c9cb6a6cadac80cfe259c4da5b2a56ce685ed"+
		"e88b2b0f7d54c1a7e230fb91c5d6241f04f3f4db89290fd059695880fa3515a3"+
		"bf811461820d937e80cd55c6bb30116ebcb774e980367c04dd6c03f71bc99eb1"+
		"f08cf9d70642c255ca07241c85b5d5a08a216e2ab2c8ccb44c5bde05bd541da5"+
		"232edb486280ed234140f14c9a8a8a308b08a9fe1e0d540b0f1f202882ea9999", 16))

// Hardcoded DSA private key for Server
var privateKey = signature.ReconstructPrivateKey(publicKey,
	large.NewIntFromString("7d169c3b371a2546c272e5ca37549c65c6a27708d87465d1b60261adf440483e", 16))

// Hardcoded DSA public key for RegistrationServer
var registrationPublicKey = signature.ReconstructPublicKey(dsaParams,
	large.NewIntFromString("1ae3fd4e9829fb464459e05bca392bec5067152fb43a569ad3c3b68bbcad84f0"+
		"ff8d31c767da3eabcfc0870d82b39568610b52f2b72b493bbede6e952c9a7fd4"+
		"4a8161e62a9046828c4a65f401b2f054ebf7376e89dab547d8a3c3d46891e78a"+
		"cfc4015713cbfb5b0b6cab0f8dfb46b891f3542046ace4cab984d5dfef4f52d4"+
		"347dc7e52f6a7ea851dda076f0ed1fef86ec6b5c2a4807149906bf8e0bf70b30"+
		"1147fea88fd95009edfbe0de8ffc1a864e4b3a24265b61a1c47a4e9307e7c84f"+
		"9b5591765b530f5859fa97b22ce9b51385d3d13088795b2f9fd0cb59357fe938"+
		"346117df2acf2bab22d942de1a70e8d5d62fc0e99d8742a0f16df94ce3a0abbb", 16))

// Handle nonce request from Client
func RequestNonce(salt, Y, P, Q, G, hash, R, S []byte) ([]byte, error) {

	if !globals.SkipRegServer {
		// Verify signed public key using hardcoded RegistrationServer public key
		valid := registrationPublicKey.Verify(hash, signature.DSASignature{
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
			return make([]byte, 0), errors.New("signed public key is invalid")
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

	// Generate user CMIX keys
	b, _ := hash2.NewCMixHash()
	baseTKey := registration.GenerateBaseKey(dsaGrp, userPublicKey, privateKey, b)
	baseRKey := registration.GenerateBaseKey(dsaGrp, userPublicKey, privateKey, sha256.New())

	// Store user information in the database
	newUser := globals.Users.NewUser(dsaGrp)
	newUser.Nonce = userNonce
	newUser.ID = userId
	newUser.PublicKey = userPublicKey
	newUser.Transmission.BaseKey = baseTKey
	newUser.Reception.BaseKey = baseRKey
	globals.Users.UpsertUser(newUser)

	// Return nonce to Client with empty error field
	return userNonce.Bytes(), nil
}

// Handle confirmation of nonce from Client
func ConfirmNonce(hash, R, S []byte) ([]byte,
	[]byte, []byte, []byte, []byte, []byte, []byte, error) {

	// Obtain the user from the database
	n := nonce.Nonce{}
	copy(n.Value[:], hash)
	user, err := globals.Users.GetUserByNonce(n)

	if err != nil {
		// Invalid nonce, return an error
		jww.ERROR.Printf("Unable to find nonce: %x", n.Bytes())
		return make([]byte, 0), make([]byte, 0), make([]byte, 0),
			make([]byte, 0), make([]byte, 0), make([]byte, 0), make([]byte, 0),
			errors.New("nonce does not exist")
	}

	// Verify nonce has not expired
	if !user.Nonce.IsValid() {
		jww.ERROR.Printf("Nonce is expired: %x", n.Bytes())
		return make([]byte, 0), make([]byte, 0), make([]byte, 0),
			make([]byte, 0), make([]byte, 0), make([]byte, 0), make([]byte, 0),
			errors.New("nonce is expired")
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
			errors.New("signed nonce is invalid")
	}

	// Concatenate Client public key byte slices
	data := make([]byte, 0)
	params := user.PublicKey.GetParams()
	data = append(data, user.PublicKey.GetKey().Bytes()...)
	data = append(data, params.GetP().Bytes()...)
	data = append(data, params.GetQ().Bytes()...)
	data = append(data, params.GetG().Bytes()...)

	// Use hardcoded Server private key to sign Client public key
	sig, err := privateKey.Sign(data, rand.Reader)
	if err != nil {
		// Unable to sign public key, return an error
		jww.ERROR.Printf("Error signing client public key: %s", err)
		return make([]byte, 0), make([]byte, 0), make([]byte, 0),
			make([]byte, 0), make([]byte, 0), make([]byte, 0), make([]byte, 0),
			errors.New("unable to sign client public key")
	}

	// Return signed Client public key to Client with empty error field
	return data, sig.R.Bytes(), sig.S.Bytes(), publicKey.GetKey().Bytes(),
		dsaParams.GetP().Bytes(), dsaParams.GetQ().Bytes(),
		dsaParams.GetG().Bytes(), nil
}
