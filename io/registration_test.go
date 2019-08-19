////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"crypto"
	"fmt"
	"gitlab.com/elixxir/crypto/cmix"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/crypto/nonce"
	"gitlab.com/elixxir/crypto/registration"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/measure"
	"os"
	"testing"
)

var serverInstance *server.Instance
var serverRSAPub *rsa.PublicKey
var serverRSAPriv *rsa.PrivateKey
var clientRSAPub *rsa.PublicKey
var clientRSAPriv *rsa.PrivateKey
var clientDHPub *cyclic.Int
var clintDHPriv *cyclic.Int
var regPrivKey *rsa.PrivateKey

func TestMain(m *testing.M) {

	primeString := "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
		"29024E088A67CC74020BBEA63B139B22514A08798E3404DD" +
		"EF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245" +
		"E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7ED" +
		"EE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3D" +
		"C2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F" +
		"83655D23DCA3AD961C62F356208552BB9ED529077096966D" +
		"670C354E4ABC9804F1746C08CA18217C32905E462E36CE3B" +
		"E39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9" +
		"DE2BCBF6955817183995497CEA956AE515D2261898FA0510" +
		"15728E5A8AACAA68FFFFFFFFFFFFFFFF"

	nid := server.GenerateId()
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2), large.NewInt(2))

	var err error

	//make client rsa key pair
	clientRSAPriv, err = rsa.GenerateKey(csprng.NewSystemRNG(), 1024)
	if err != nil {
		panic(fmt.Sprintf("Could not generate node private key: %+v", err))
	}

	clientRSAPub = clientRSAPriv.GetPublic()

	//make client DH key
	clintDHPriv = grp.RandomCoprime(grp.NewInt(1))
	clientDHPub = grp.ExpG(clintDHPriv, grp.NewInt(1))

	//make registration rsa key pair
	regPrivKey, err = rsa.GenerateKey(csprng.NewSystemRNG(), 1024)
	if err != nil {
		panic(fmt.Sprintf("Could not generate registration private key: %+v", err))
	}

	//make server rsa key pair
	serverRSAPriv, err = rsa.GenerateKey(csprng.NewSystemRNG(), 1024)
	if err != nil {
		panic(fmt.Sprintf("Could not generate node private key: %+v", err))
	}

	serverRSAPub = serverRSAPriv.GetPublic()

	def := server.Definition{
		CmixGroup: grp,
		Nodes: []server.Node{
			{
				ID: nid,
			},
		},
		ID:              nid,
		UserRegistry:    &globals.UserMap{},
		ResourceMonitor: &measure.ResourceMonitor{},
		PrivateKey:      serverRSAPriv,
		PublicKey:       serverRSAPub,
	}

	def.Permissioning.PublicKey = regPrivKey.GetPublic()

	serverInstance = server.CreateServerInstance(&def)

	os.Exit(m.Run())
}

// TODO: How should the public key be retrieved?
// Is it from the Permissioning.Paths.Cert?
// Perhaps Paths object should get a GetPublicKey Method?
// Test request nonce
func TestRequestNonce(t *testing.T) {
	rng := csprng.NewSystemRNG()
	salt := cmix.NewSalt(rng, 32)

	clientRSAPubKeyPEM := rsa.CreatePublicKeyPem(clientRSAPub)

	//sign the client's RSA key by registration
	sha := crypto.SHA256
	h := sha.New()
	h.Write(clientRSAPubKeyPEM)
	data := h.Sum(nil)

	sigReg, err := rsa.Sign(csprng.NewSystemRNG(), regPrivKey, sha, data, nil)

	if err != nil {
		t.Errorf("Could not sign client's RSA key with registration's "+
			"key: %+v", err)
	}

	//sign the client's DH key with client's RSA key pair
	h.Reset()
	h.Write(clientDHPub.Bytes())
	data = h.Sum(nil)

	sigClient, err := rsa.Sign(csprng.NewSystemRNG(), clientRSAPriv, sha, data, nil)

	if err != nil {
		t.Errorf("COuld not sign client's DH key with client RSA "+
			"key: %+v", err)
	}

	result, _, err2 := RequestNonce(serverInstance,
		salt, string(clientRSAPubKeyPEM), clientDHPub.Bytes(), sigReg, sigClient)

	if result == nil || err2 != nil {
		t.Errorf("Error in RequestNonce: %+v", err2)
	}
}

// Test request nonce with invalid signature from registration
func TestRequestNonce_BadRegSignature(t *testing.T) {
	rng := csprng.NewSystemRNG()
	salt := cmix.NewSalt(rng, 32)

	clientRSAPubKeyPEM := rsa.CreatePublicKeyPem(clientRSAPub)

	//dont sign the client's RSA key by registration
	sha := crypto.SHA256
	h := sha.New()

	sigReg := make([]byte, 69)

	//sign the client's DH key with client's RSA key pair
	h.Reset()
	h.Write(clientDHPub.Bytes())
	data := h.Sum(nil)

	sigClient, err := rsa.Sign(csprng.NewSystemRNG(), clientRSAPriv, sha, data, nil)

	if err != nil {
		t.Errorf("COuld not sign client's DH key with client RSA "+
			"key: %+v", err)
	}

	_, _, err2 := RequestNonce(serverInstance,
		salt, string(clientRSAPubKeyPEM), clientDHPub.Bytes(), sigReg, sigClient)

	if err2 == nil {
		t.Errorf("Error in RequestNonce, did not fail with bad "+
			"registartion signature: %+v", err2)
	}
}

// Test request nonce with invalid signature from client
func TestRequestNonce_BadClientSignature(t *testing.T) {
	rng := csprng.NewSystemRNG()
	salt := cmix.NewSalt(rng, 32)

	clientRSAPubKeyPEM := rsa.CreatePublicKeyPem(clientRSAPub)

	//sign the client's RSA key by registration
	sha := crypto.SHA256
	h := sha.New()
	h.Write(clientRSAPubKeyPEM)
	data := h.Sum(nil)

	sigReg, err := rsa.Sign(csprng.NewSystemRNG(), regPrivKey, sha, data, nil)

	if err != nil {
		t.Errorf("Could not sign client's RSA key with registration's "+
			"key: %+v", err)
	}

	//dont sign the client's DH key with client's RSA key pair
	sigClient := make([]byte, 42)

	_, _, err2 := RequestNonce(serverInstance,
		salt, string(clientRSAPubKeyPEM), clientDHPub.Bytes(), sigReg, sigClient)

	if err2 == nil {
		t.Errorf("Error in RequestNonce, did not fail with bad "+
			"registartion signature: %+v", err2)
	}
}

// Test confirm nonce
func TestConfirmNonce(t *testing.T) {
	//make new user
	user := serverInstance.GetUserRegistry().NewUser(serverInstance.GetGroup())
	user.Nonce, _ = nonce.NewNonce(nonce.RegistrationTTL)
	user.IsRegistered = false
	user.RsaPublicKey = clientRSAPub
	user.ID = registration.GenUserID(clientRSAPub, []byte{69})
	serverInstance.GetUserRegistry().UpsertUser(user)

	//hash and sign nonce
	sha := crypto.SHA256

	h := sha.New()
	h.Write(user.Nonce.Bytes())
	data := h.Sum(nil)

	sign, err := rsa.Sign(csprng.NewSystemRNG(), clientRSAPriv, sha, data, nil)
	if sign == nil || err != nil {
		t.Errorf("Error signing data")
	}

	//call confirm
	_, err2 := ConfirmRegistration(serverInstance, user.ID.Bytes(), sign)
	if err2 != nil {
		t.Errorf("Error in ConfirmNonce: %+v", err2)
	}

	regUser, err := serverInstance.GetUserRegistry().GetUser(user.ID)

	if err != nil {
		t.Errorf("User could not be found: %+v", err)
	}

	if !regUser.IsRegistered {
		t.Errorf("User's registation was not sucesfully confirmed: %+v", regUser)
	}
}

/*
// Test confirm nonce that doesn't exist
func TestConfirmNonce_NonExistant(t *testing.T) {
	user := serverInstance.GetUserRegistry().NewUser(serverInstance.GetGroup())
	user.Nonce, _ = nonce.NewNonce(nonce.RegistrationTTL)

	rng := csprng.NewSystemRNG()
	user.PublicKey = clientRSAPub

	sign, err := clientRSAPriv.Sign(user.Nonce.Bytes(), rng)
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
	user.Nonce, _ = nonce.NewNonce(1)
	serverInstance.GetUserRegistry().UpsertUser(user)

	rng := csprng.NewSystemRNG()
	user.PublicKey = clientRSAPub

	sign, err := clientRSAPriv.Sign(user.Nonce.Bytes(), rng)
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
	user.Nonce, _ = nonce.NewNonce(nonce.RegistrationTTL)
	serverInstance.GetUserRegistry().UpsertUser(user)

	_, _, _, _, _, _, _, err := ConfirmRegistration(serverInstance,
		user.Nonce.Bytes(), make([]byte, 0),
		make([]byte, 0))
	if err == nil {
		t.Errorf("ConfirmNonce: Expected bad signature!")
	}
}

func initConfGroups(grp *cyclic.Group) conf.Groups {

	primeString := grp.GetP().TextVerbose(16, 0)
	smallprime := grp.GetQ().TextVerbose(16, 0)
	generator := grp.GetG().TextVerbose(16, 0)

	cmixMap := map[string]string{
		"prime":      primeString,
		"smallprime": smallprime,
		"generator":  generator,
	}

	grps := conf.Groups{
		CMix: cmixMap,
	}

	return grps
}*/
