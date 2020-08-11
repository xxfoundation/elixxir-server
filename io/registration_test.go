///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package io

import (
	"crypto"
	"fmt"
	"gitlab.com/elixxir/comms/testkeys"
	"gitlab.com/elixxir/crypto/cmix"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/crypto/xx"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/utils"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/state"
	"gitlab.com/elixxir/server/testUtil"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/nonce"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"os"
	"testing"
	"time"
)

var serverInstance *internal.Instance
var clientRSAPub *rsa.PublicKey
var clientRSAPriv *rsa.PrivateKey
var clientDHPub *cyclic.Int
var regPrivKey *rsa.PrivateKey
var nodeId *id.ID

func setup(t interface{}) (*internal.Instance, *rsa.PublicKey, *rsa.PrivateKey, *cyclic.Int, *rsa.PrivateKey, *id.ID, string) {
	switch v := t.(type) {
	case *testing.T:
	case *testing.M:
		break
	default:
		panic(fmt.Sprintf("Cannot use outside of test environment; %+v", v))
	}

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

	cert, _ := utils.ReadFile(testkeys.GetNodeCertPath())
	key, _ := utils.ReadFile(testkeys.GetNodeKeyPath())

	nid := internal.GenerateId(t)
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2))

	var err error

	//make client rsa key pair
	cRsaPriv, err := rsa.GenerateKey(csprng.NewSystemRNG(), 1024)
	if err != nil {
		panic(fmt.Sprintf("Could not generate node private key: %+v", err))
	}

	cRsaPub := cRsaPriv.GetPublic()

	//make client DH key
	clintDHPriv := grp.RandomCoprime(grp.NewInt(1))
	cDhPub := grp.ExpG(clintDHPriv, grp.NewInt(1))

	//make registration rsa key pair
	regPKey, err := rsa.GenerateKey(csprng.NewSystemRNG(), 1024)
	if err != nil {
		panic(fmt.Sprintf("Could not generate registration private key: %+v", err))
	}

	//make server rsa key pair
	serverRSAPriv, err := rsa.GenerateKey(csprng.NewSystemRNG(), 1024)
	if err != nil {
		panic(fmt.Sprintf("Could not generate node private key: %+v", err))
	}

	serverRSAPub := serverRSAPriv.GetPublic()
	nodeAddr := fmt.Sprintf("0.0.0.0:%d", 7000+rand.Intn(1000))

	def := internal.Definition{
		ID:              nid,
		UserRegistry:    &globals.UserMap{},
		ResourceMonitor: &measure.ResourceMonitor{},
		PrivateKey:      serverRSAPriv,
		PublicKey:       serverRSAPub,
		TlsCert:         cert,
		TlsKey:          key,
		FullNDF:         testUtil.NDF,
		PartialNDF:      testUtil.NDF,
		Address:         nodeAddr,
	}

	def.Permissioning.PublicKey = regPKey.GetPublic()
	nodeIDs := make([]*id.ID, 0)
	nodeIDs = append(nodeIDs, nid)
	def.Gateway.ID = &id.TempGateway

	mach := state.NewTestMachine(dummyStates, current.PRECOMPUTING, t)

	instance, _ := internal.CreateServerInstance(&def, NewImplementation,
		mach, "1.1.0")

	return instance, cRsaPub, cRsaPriv, cDhPub, regPKey, nid, nodeAddr
}

func TestMain(m *testing.M) { // TODO: TestMain is bad make this go away
	serverInstance, clientRSAPub, clientRSAPriv, clientDHPub, regPrivKey, nodeId, _ = setup(m)
	os.Exit(m.Run())
}

// Test request nonce with good auth boolean but bad ID
func TestRequestNonceFailAuthId(t *testing.T) {
	newID := id.NewIdFromString("420blazeit", id.Node, t)

	// The incorrect ID here is the crux of the test
	gwHost, err := connect.NewHost(newID, "", make([]byte, 0), false, true)
	if err != nil {
		t.Errorf("Unable to create gateway host: %+v", err)
	}

	_, _, err2 := RequestNonce(serverInstance,
		make([]byte, 0), "", clientDHPub.Bytes(), make([]byte, 0),
		make([]byte, 0), &connect.Auth{
			IsAuthenticated: true, // True for this test, we want bad sender ID
			Sender:          gwHost,
		})

	if !connect.IsAuthError(err2) {
		t.Errorf("Expected auth error in RequestNonce: %+v", err2)
	}
}

// Test request nonce with bad auth boolean but good ID
func TestRequestNonceFailAuth(t *testing.T) {
	gwID := nodeId.DeepCopy()
	gwID.SetType(id.Gateway)

	gwHost, err := connect.NewHost(gwID, "", make([]byte, 0), false, true)
	if err != nil {
		t.Errorf("Unable to create gateway host: %+v", err)
	}

	_, _, err2 := RequestNonce(serverInstance,
		make([]byte, 0), "", clientDHPub.Bytes(), make([]byte, 0),
		make([]byte, 0), &connect.Auth{
			IsAuthenticated: false, // This is the crux of the test
			Sender:          gwHost,
		})

	if !connect.IsAuthError(err2) {
		t.Errorf("Expected auth error in RequestNonce: %+v", err2)
	}
}

// Test request nonce happy path
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

	gwHost, err := connect.NewHost(&id.TempGateway, "", make([]byte, 0), false, true)
	if err != nil {
		t.Errorf("Unable to create gateway host: %+v", err)
	}

	result, _, err2 := RequestNonce(serverInstance, salt,
		string(clientRSAPubKeyPEM), clientDHPub.Bytes(), sigReg, sigClient,
		&connect.Auth{
			IsAuthenticated: true,
			Sender:          gwHost,
		})

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

	gwID := nodeId.DeepCopy()
	gwID.SetType(id.Gateway)

	gwHost, err := connect.NewHost(gwID, "", make([]byte, 0), false, true)
	if err != nil {
		t.Errorf("Unable to create gateway host: %+v", err)
	}
	_, _, err2 := RequestNonce(serverInstance, salt, string(clientRSAPubKeyPEM),
		clientDHPub.Bytes(), sigReg, sigClient,
		&connect.Auth{
			IsAuthenticated: true,
			Sender:          gwHost,
		})

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

	gwID := nodeId.DeepCopy()
	gwID.SetType(id.Gateway)

	gwHost, err := connect.NewHost(gwID, "", make([]byte, 0), false, true)
	if err != nil {
		t.Errorf("Unable to create gateway host: %+v", err)
	}
	_, _, err2 := RequestNonce(serverInstance, salt, string(clientRSAPubKeyPEM),
		clientDHPub.Bytes(), sigReg, sigClient,
		&connect.Auth{
			IsAuthenticated: true,
			Sender:          gwHost,
		})

	if err2 == nil {
		t.Errorf("Error in RequestNonce, did not fail with bad "+
			"registartion signature: %+v", err2)
	}
}

// Test confirm nonce
func TestConfirmRegistration(t *testing.T) {
	//make new user
	user := serverInstance.GetUserRegistry().NewUser(serverInstance.GetConsensus().GetCmixGroup())
	user.Nonce, _ = nonce.NewNonce(nonce.RegistrationTTL)
	user.IsRegistered = false
	user.RsaPublicKey = clientRSAPub
	salt := cmix.NewSalt(csprng.NewSystemRNG(), 32)

	userID, err := xx.NewID(clientRSAPub, salt, id.User)
	if err != nil {
		t.Errorf("Error creating new user ID: %+v", err)
	}
	user.ID = userID

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

	gwHost, err := connect.NewHost(&id.TempGateway, "", make([]byte, 0), false, true)
	if err != nil {
		t.Errorf("Unable to create gateway host: %+v", err)
	}

	//call confirm
	_, _, err2 := ConfirmRegistration(serverInstance, user.ID, sign,
		&connect.Auth{
			IsAuthenticated: true,
			Sender:          gwHost,
		})
	if err2 != nil {
		t.Errorf("Error in ConfirmRegistration: %+v", err2)
	}

	regUser, err := serverInstance.GetUserRegistry().GetUser(user.ID, serverInstance.GetConsensus().GetCmixGroup())

	if err != nil {
		t.Errorf("User could not be found: %+v", err)
	}

	if !regUser.IsRegistered {
		t.Errorf("User's registation was not sucesfully confirmed: %+v", regUser)
	}
}

// Test confirm nonce with bad auth boolean but good ID
func TestConfirmRegistrationFailAuth(t *testing.T) {
	user := serverInstance.GetUserRegistry().NewUser(serverInstance.GetConsensus().GetCmixGroup())
	user.Nonce, _ = nonce.NewNonce(nonce.RegistrationTTL)

	user.RsaPublicKey = clientRSAPub

	//hash and sign nonce
	sha := crypto.SHA256

	h := sha.New()
	h.Write(user.Nonce.Bytes())
	data := h.Sum(nil)

	sign, err := rsa.Sign(csprng.NewSystemRNG(), clientRSAPriv, sha, data, nil)
	if sign == nil || err != nil {
		t.Errorf("Error signing data")
	}

	gwID := nodeId.DeepCopy()
	gwID.SetType(id.Gateway)

	gwHost, err := connect.NewHost(gwID, "", make([]byte, 0), false, true)
	if err != nil {
		t.Errorf("Unable to create gateway host: %+v", err)
	}

	_, _, err2 := ConfirmRegistration(serverInstance, user.ID, sign,
		&connect.Auth{
			IsAuthenticated: false, // This is the crux of the test
			Sender:          gwHost,
		})
	if err2 == nil {
		t.Errorf("ConfirmRegistration: Expected unexistant nonce")
	}
}

// Test confirm nonce with bad auth boolean but good ID
func TestConfirmRegistrationFailAuthId(t *testing.T) {
	user := serverInstance.GetUserRegistry().NewUser(serverInstance.GetConsensus().GetCmixGroup())
	user.Nonce, _ = nonce.NewNonce(nonce.RegistrationTTL)

	user.RsaPublicKey = clientRSAPub

	//hash and sign nonce
	sha := crypto.SHA256

	h := sha.New()
	h.Write(user.Nonce.Bytes())
	data := h.Sum(nil)

	sign, err := rsa.Sign(csprng.NewSystemRNG(), clientRSAPriv, sha, data, nil)
	if sign == nil || err != nil {
		t.Errorf("Error signing data")
	}

	newID := id.NewIdFromString("420blzit", id.Node, t)

	// The incorrect ID here is the crux of the test
	gwHost, err := connect.NewHost(newID, "", make([]byte, 0), false, true)
	if err != nil {
		t.Errorf("Unable to create gateway host: %+v", err)
	}

	_, _, err2 := ConfirmRegistration(serverInstance, user.ID, sign,
		&connect.Auth{
			IsAuthenticated: true, // True for this test, we want bad sender ID
			Sender:          gwHost,
		})
	if err2 == nil {
		t.Errorf("ConfirmRegistration: Expected unexistant nonce")
	}
}

// Test confirm nonce that doesn't exist
func TestConfirmRegistration_NonExistant(t *testing.T) {
	user := serverInstance.GetUserRegistry().NewUser(serverInstance.GetConsensus().GetCmixGroup())
	user.Nonce, _ = nonce.NewNonce(nonce.RegistrationTTL)

	user.RsaPublicKey = clientRSAPub

	//hash and sign nonce
	sha := crypto.SHA256

	h := sha.New()
	h.Write(user.Nonce.Bytes())
	data := h.Sum(nil)

	sign, err := rsa.Sign(csprng.NewSystemRNG(), clientRSAPriv, sha, data, nil)
	if sign == nil || err != nil {
		t.Errorf("Error signing data")
	}

	gwID := nodeId.DeepCopy()
	gwID.SetType(id.Gateway)

	gwHost, err := connect.NewHost(gwID, "", make([]byte, 0), false, true)
	if err != nil {
		t.Errorf("Unable to create gateway host: %+v", err)
	}

	_, _, err2 := ConfirmRegistration(serverInstance, user.ID, sign,
		&connect.Auth{
			IsAuthenticated: true,
			Sender:          gwHost,
		})
	if err2 == nil {
		t.Errorf("ConfirmRegistration: Expected unexistant nonce")
	}
}

// Test confirm nonce expired
func TestConfirmRegistration_Expired(t *testing.T) {
	user := serverInstance.GetUserRegistry().NewUser(serverInstance.GetConsensus().GetCmixGroup())
	user.Nonce, _ = nonce.NewNonce(1)
	serverInstance.GetUserRegistry().UpsertUser(user)

	user.RsaPublicKey = clientRSAPub

	//hash and sign nonce
	sha := crypto.SHA256

	h := sha.New()
	h.Write(user.Nonce.Bytes())
	data := h.Sum(nil)

	sign, err := rsa.Sign(csprng.NewSystemRNG(), clientRSAPriv, sha, data, nil)
	if sign == nil || err != nil {
		t.Errorf("Error signing data")
	}

	// Wait for nonce to expire
	wait := time.After(time.Duration(2) * time.Second)
	select {
	case <-wait:
	}

	gwID := nodeId.DeepCopy()
	gwID.SetType(id.Gateway)

	gwHost, err := connect.NewHost(gwID, "", make([]byte, 0), false, true)
	if err != nil {
		t.Errorf("Unable to create gateway host: %+v", err)
	}

	_, _, err2 := ConfirmRegistration(serverInstance, user.ID, sign,
		&connect.Auth{
			IsAuthenticated: true,
			Sender:          gwHost,
		})
	if err2 == nil {
		t.Errorf("ConfirmRegistration: Expected expired nonce")
	}
}

// Test confirm nonce with invalid signature
func TestConfirmRegistration_BadSignature(t *testing.T) {
	user := serverInstance.GetUserRegistry().NewUser(serverInstance.GetConsensus().GetCmixGroup())
	user.Nonce, _ = nonce.NewNonce(nonce.RegistrationTTL)
	serverInstance.GetUserRegistry().UpsertUser(user)
	user.RsaPublicKey = clientRSAPub

	gwID := nodeId.DeepCopy()
	gwID.SetType(id.Gateway)

	gwHost, err := connect.NewHost(gwID, "", make([]byte, 0), false, true)
	if err != nil {
		t.Errorf("Unable to create gateway host: %+v", err)
	}

	_, _, err = ConfirmRegistration(serverInstance, user.ID, []byte("test"),
		&connect.Auth{
			IsAuthenticated: true,
			Sender:          gwHost,
		})
	if err == nil {
		t.Errorf("ConfirmRegistration: Expected bad signature!")
	}
}
