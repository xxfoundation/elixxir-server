////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/elixxir/crypto/cmix"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/crypto/nonce"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/conf"
	"os"
	"testing"
	"time"
)

var serverInstance *server.Instance
var dsaParams = signature.GetDefaultDSAParams()

func TestMain(m *testing.M) {
	grp := cyclic.NewGroup(dsaParams.GetP(), dsaParams.GetG(), dsaParams.GetQ())

	nid := server.GenerateId()
	params := conf.Params{
		Groups: conf.Groups{
			CMix: grp,
		},
		NodeIDs: []string{nid.String()},
	}

	serverInstance = server.CreateServerInstance(params, &globals.UserMap{})
	os.Exit(m.Run())
}

// Test request nonce
func TestRequestNonce(t *testing.T) {
	regPrivKey := signature.ReconstructPrivateKey(serverInstance.GetRegPubKey(),
		large.NewIntFromString("dab0febfab103729077ad4927754f6390e366fdf4c58e8d40dadb3e94c444b54", 16))
	rng := csprng.NewSystemRNG()
	privKey := dsaParams.PrivateKeyGen(rng)
	pubKey := privKey.PublicKeyGen()
	salt := cmix.NewSalt(rng, 32)

	hash := append(pubKey.GetKey().Bytes(), dsaParams.GetP().Bytes()...)
	hash = append(hash, dsaParams.GetQ().Bytes()...)
	hash = append(hash, dsaParams.GetG().Bytes()...)

	sign, err := regPrivKey.Sign(hash, rng)
	if sign == nil || err != nil {
		t.Errorf("Error signing data: %v", err.Error())
	}

	result, err2 := RequestNonce(serverInstance,
		salt,
		pubKey.GetKey().Bytes(),
		dsaParams.GetP().Bytes(),
		dsaParams.GetQ().Bytes(),
		dsaParams.GetG().Bytes(),
		hash,
		sign.R.Bytes(),
		sign.S.Bytes())

	if result == nil || err2 != nil {
		t.Errorf("Error in RequestNonce")
	}
}

// Test request nonce with invalid signature
func TestRequestNonce_BadSignature(t *testing.T) {
	rng := csprng.NewSystemRNG()
	privKey := dsaParams.PrivateKeyGen(rng)
	pubKey := privKey.PublicKeyGen()
	salt := cmix.NewSalt(rng, 32)
	regPrivKey := dsaParams.PrivateKeyGen(rng)

	hash := append(pubKey.GetKey().Bytes(), dsaParams.GetP().Bytes()...)
	hash = append(hash, dsaParams.GetQ().Bytes()...)
	hash = append(hash, dsaParams.GetG().Bytes()...)

	sign, err := regPrivKey.Sign(hash, rng)
	if sign == nil || err != nil {
		t.Errorf("Error signing data")
	}

	_, err2 := RequestNonce(serverInstance,
		salt,
		pubKey.GetKey().Bytes(),
		dsaParams.GetP().Bytes(),
		dsaParams.GetQ().Bytes(),
		dsaParams.GetG().Bytes(),
		hash,
		sign.R.Bytes(),
		sign.S.Bytes())

	if err2 == nil {
		t.Errorf("Expected error in RequestNonce")
	}
}

// Test confirm nonce
func TestConfirmNonce(t *testing.T) {
	user := serverInstance.GetUserRegistry().NewUser(serverInstance.GetGroup())
	user.Nonce = nonce.NewNonce(nonce.RegistrationTTL)
	serverInstance.GetUserRegistry().UpsertUser(user)

	rng := csprng.NewSystemRNG()
	privKey := dsaParams.PrivateKeyGen(rng)
	pubKey := privKey.PublicKeyGen()
	user.PublicKey = pubKey

	sign, err := privKey.Sign(user.Nonce.Bytes(), rng)
	if sign == nil || err != nil {
		t.Errorf("Error signing data")
	}

	_, _, _, _, _, _, _, err2 := ConfirmRegistration(serverInstance,
		user.Nonce.Bytes(), sign.R.Bytes(), sign.S.Bytes())
	if err2 != nil {
		t.Errorf("Error in ConfirmNonce")
	}
}

// Test confirm nonce that doesn't exist
func TestConfirmNonce_NonExistant(t *testing.T) {
	user := serverInstance.GetUserRegistry().NewUser(serverInstance.GetGroup())
	user.Nonce = nonce.NewNonce(nonce.RegistrationTTL)

	rng := csprng.NewSystemRNG()
	privKey := dsaParams.PrivateKeyGen(rng)
	pubKey := privKey.PublicKeyGen()
	user.PublicKey = pubKey

	sign, err := privKey.Sign(user.Nonce.Bytes(), rng)
	if sign == nil || err != nil {
		t.Errorf("Error signing data")
	}

	_, _, _, _, _, _, _, err2 := ConfirmRegistration(serverInstance,
		user.Nonce.Bytes(), sign.R.Bytes(), sign.S.Bytes())
	if err2 == nil {
		t.Errorf("ConfirmNonce: Expected unexistant nonce")
	}
}

// Test confirm nonce expired
func TestConfirmNonce_Expired(t *testing.T) {
	user := serverInstance.GetUserRegistry().NewUser(serverInstance.GetGroup())
	user.Nonce = nonce.NewNonce(1)
	serverInstance.GetUserRegistry().UpsertUser(user)

	rng := csprng.NewSystemRNG()
	privKey := dsaParams.PrivateKeyGen(rng)
	pubKey := privKey.PublicKeyGen()
	user.PublicKey = pubKey

	sign, err := privKey.Sign(user.Nonce.Bytes(), rng)
	if sign == nil || err != nil {
		t.Errorf("Error signing data")
	}

	// Wait for nonce to expire
	wait := time.After(time.Duration(2) * time.Second)
	select {
	case <-wait:
	}

	_, _, _, _, _, _, _, err2 := ConfirmRegistration(serverInstance,
		user.Nonce.Bytes(), sign.R.Bytes(), sign.S.Bytes())
	if err2 == nil {
		t.Errorf("ConfirmNonce: Expected expired nonce")
	}
}

// Test confirm nonce with invalid signature
func TestConfirmNonce_BadSignature(t *testing.T) {
	user := serverInstance.GetUserRegistry().NewUser(serverInstance.GetGroup())
	user.Nonce = nonce.NewNonce(nonce.RegistrationTTL)
	serverInstance.GetUserRegistry().UpsertUser(user)

	_, _, _, _, _, _, _, err := ConfirmRegistration(serverInstance,
		user.Nonce.Bytes(), make([]byte, 0),
		make([]byte, 0))
	if err == nil {
		t.Errorf("ConfirmNonce: Expected bad signature!")
	}
}
