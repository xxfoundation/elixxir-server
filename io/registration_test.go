////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"crypto"
	cryptoRand "crypto/rand"
	gorsa "crypto/rsa"
	"fmt"
	"github.com/golang/protobuf/proto"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/testkeys"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/registration"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/internal/state"
	"gitlab.com/elixxir/server/storage"
	"gitlab.com/elixxir/server/testUtil"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/utils"
	"math/rand"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Happy path
func TestRequestClientKey(t *testing.T) {
	instance, userRsaPub, userRsaPriv, userDhPrivKey, userDhPubKey, clientRegistrarPrivKey, _, _ := setup(t)

	// Construct host for auth
	gwHost, err := connect.NewHost(&id.TempGateway, "", make([]byte, 0), connect.GetDefaultHostParams())
	if err != nil {
		t.Errorf("Unable to create gateway host: %+v", err)
	}

	// Construct auth
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          gwHost,
	}

	// Generate a pre-canned time for consistent testing
	testTime, err := time.Parse(time.RFC3339,
		"2012-12-21T22:08:41+00:00")
	if err != nil {
		t.Fatalf("RequestNonce error: "+
			"Could not parse precanned time: %v", err.Error())
	}
	// Convert public key to PEM
	clientRSAPubKeyPEM := rsa.CreatePublicKeyPem(userRsaPub)

	// Sign timestamp
	sigReg, err := registration.SignWithTimestamp(csprng.NewSystemRNG(),
		clientRegistrarPrivKey, testTime.UnixNano(), string(clientRSAPubKeyPEM))
	if err != nil {
		t.Errorf("Could not sign client's RSA key with registration's "+
			"key: %+v", err)
	}

	regConfirm := &pb.ClientRegistrationConfirmation{
		RSAPubKey: string(clientRSAPubKeyPEM),
		Timestamp: testTime.UnixNano(),
	}

	regConfirmBytes, err := proto.Marshal(regConfirm)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	salt := make([]byte, 32)
	copy(salt, "saltData")
	// Construct request
	request := &pb.ClientKeyRequest{
		Salt: salt,
		ClientTransmissionConfirmation: &pb.SignedRegistrationConfirmation{
			ClientRegistrationConfirmation: regConfirmBytes,
			RegistrarSignature:             &messages.RSASignature{Signature: sigReg},
		},
		RegistrationTimestamp: testTime.UnixNano(),
		ClientDHPubKey:        userDhPubKey.Bytes(),
	}

	// Marshal request
	requestBytes, err := proto.Marshal(request)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	for _, useSha := range []bool{false, true} {
		t.Run(fmt.Sprintf("TestRequestClientKey[useSha=%+v]", useSha), func(t *testing.T) {
			// Hash request
			opts := rsa.NewDefaultOptions()
			if useSha {
				opts.Hash = crypto.SHA256
			}
			h := opts.Hash.New()
			h.Write(requestBytes)
			hashedData := h.Sum(nil)

			// Sign the request with the user's private key
			requestSig, err := rsa.Sign(csprng.NewSystemRNG(), userRsaPriv, opts.Hash, hashedData, opts)
			if err != nil {
				t.Fatalf("Sign error: %v", err)
			}

			// Construct signed request
			signedRequest := &pb.SignedClientKeyRequest{
				ClientKeyRequest:          requestBytes,
				ClientKeyRequestSignature: &messages.RSASignature{Signature: requestSig},
				UseSHA:                    useSha,
			}

			// Request key with mock message
			signedKeyResp, err := RequestClientKey(instance, signedRequest, auth)
			if err != nil {
				t.Fatalf("RequestClientKey error: %v", err)
			}

			// Unmarshal response
			keyResponse := &pb.ClientKeyResponse{}
			err = proto.Unmarshal(signedKeyResp.KeyResponse, keyResponse)
			if err != nil {
				t.Fatalf("Failed to unmarshal message: %v", err)
			}

			// Construct the session key
			h.Reset()
			grp := instance.GetNetworkStatus().GetCmixGroup()
			nodeDHPub := grp.NewIntFromBytes(keyResponse.NodeDHPubKey)
			sessionKey := registration.GenerateBaseKey(grp,
				nodeDHPub, userDhPrivKey, h)

			// Verify the HMAC
			h.Reset()
			if !registration.VerifyClientHMAC(sessionKey.Bytes(), keyResponse.EncryptedClientKey,
				opts.Hash.New, keyResponse.EncryptedClientKeyHMAC) {
				t.Fatalf("Failed to verify client HMAC")
			}

		})
	}
}

// Error path: bad auth passed
func TestRequestClientKey_BadAuth(t *testing.T) {
	instance, _, _, _, _, _, _, _ := setup(t)

	newID := id.NewIdFromString("Jonathan", id.Node, t)

	// The incorrect ID here is the crux of the test
	gwHost, err := connect.NewHost(newID, "", make([]byte, 0), connect.GetDefaultHostParams())
	if err != nil {
		t.Errorf("Unable to create gateway host: %+v", err)
	}

	_, err = RequestClientKey(instance, &pb.SignedClientKeyRequest{}, &connect.Auth{
		IsAuthenticated: true, // True for this test, we want bad sender ID
		Sender:          gwHost,
	})

	if !connect.IsAuthError(err) {
		t.Errorf("Expected auth error in RequestNonce: %+v", err)
	}
}

// Error path: In testing setup, construct registrar signature w/
// an unexpected private key, which should cause a verification failure.
func TestRequestClientKey_BadClientRegistrarSignature(t *testing.T) {
	instance, userRsaPub, _, _, _, _, _, _ := setup(t)

	// Construct host for auth
	gwHost, err := connect.NewHost(&id.TempGateway, "", make([]byte, 0), connect.GetDefaultHostParams())
	if err != nil {
		t.Errorf("Unable to create gateway host: %+v", err)
	}

	// Construct auth
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          gwHost,
	}

	// Generate a pre-canned time for consistent testing
	testTime, err := time.Parse(time.RFC3339,
		"2012-12-21T22:08:41+00:00")
	if err != nil {
		t.Fatalf("RequestNonce error: "+
			"Could not parse precanned time: %v", err.Error())
	}
	// Convert public key to PEM
	clientRSAPubKeyPEM := rsa.CreatePublicKeyPem(userRsaPub)

	// Sign timestamp incorrectly with server private key
	sigReg, err := registration.SignWithTimestamp(csprng.NewSystemRNG(),
		instance.GetPrivKey(), testTime.UnixNano(), string(clientRSAPubKeyPEM))
	if err != nil {
		t.Errorf("Could not sign client's RSA key with registration's "+
			"key: %+v", err)
	}

	regConfirm := &pb.ClientRegistrationConfirmation{
		RSAPubKey: string(clientRSAPubKeyPEM),
		Timestamp: testTime.UnixNano(),
	}

	regConfirmBytes, err := proto.Marshal(regConfirm)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	// Construct request
	request := &pb.ClientKeyRequest{
		ClientTransmissionConfirmation: &pb.SignedRegistrationConfirmation{
			ClientRegistrationConfirmation: regConfirmBytes,
			RegistrarSignature:             &messages.RSASignature{Signature: sigReg},
		},
		RegistrationTimestamp: testTime.UnixNano(),
	}

	requestBytes, err := proto.Marshal(request)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	// Construct signed request
	signedRequest := &pb.SignedClientKeyRequest{
		ClientKeyRequest: requestBytes,
	}

	_, err = RequestClientKey(instance, signedRequest, auth)
	if err == nil ||
		!strings.HasSuffix(err.Error(), gorsa.ErrVerification.Error()) {
		t.Errorf("Expected error case: " +
			"Registration signature should have failed.")
	}

}

func TestRequestClientKey_BadClientSignature(t *testing.T) {
	instance, userRsaPub, _, _, userDhPubKey, clientRegistrarPrivKey, _, _ := setup(t)

	// Construct host for auth
	gwHost, err := connect.NewHost(&id.TempGateway, "", make([]byte, 0), connect.GetDefaultHostParams())
	if err != nil {
		t.Errorf("Unable to create gateway host: %+v", err)
	}

	// Construct auth
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          gwHost,
	}

	// Generate a pre-canned time for consistent testing
	testTime, err := time.Parse(time.RFC3339,
		"2012-12-21T22:08:41+00:00")
	if err != nil {
		t.Fatalf("RequestNonce error: "+
			"Could not parse precanned time: %v", err.Error())
	}
	// Convert public key to PEM
	clientRSAPubKeyPEM := rsa.CreatePublicKeyPem(userRsaPub)

	// Sign timestamp
	sigReg, err := registration.SignWithTimestamp(csprng.NewSystemRNG(),
		clientRegistrarPrivKey, testTime.UnixNano(), string(clientRSAPubKeyPEM))
	if err != nil {
		t.Errorf("Could not sign client's RSA key with registration's "+
			"key: %+v", err)
	}

	regConfirm := &pb.ClientRegistrationConfirmation{
		RSAPubKey: string(clientRSAPubKeyPEM),
		Timestamp: testTime.UnixNano(),
	}

	regConfirmBytes, err := proto.Marshal(regConfirm)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	salt := make([]byte, 32)
	copy(salt, "saltData")
	// Construct request
	request := &pb.ClientKeyRequest{
		Salt: salt,
		ClientTransmissionConfirmation: &pb.SignedRegistrationConfirmation{
			ClientRegistrationConfirmation: regConfirmBytes,
			RegistrarSignature:             &messages.RSASignature{Signature: sigReg},
		},
		RegistrationTimestamp: testTime.UnixNano(),
		ClientDHPubKey:        userDhPubKey.Bytes(),
	}

	// Marshal request
	requestBytes, err := proto.Marshal(request)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	// Hash request
	opts := rsa.NewDefaultOptions()
	h := opts.Hash.New()
	h.Write(requestBytes)
	hashedData := h.Sum(nil)

	// Sign the request incorrectly with the registrar's private key
	requestSig, err := rsa.Sign(csprng.NewSystemRNG(), clientRegistrarPrivKey,
		opts.Hash, hashedData, opts)
	if err != nil {
		t.Fatalf("Sign error: %v", err)
	}

	// Construct signed request
	signedRequest := &pb.SignedClientKeyRequest{
		ClientKeyRequest:          requestBytes,
		ClientKeyRequestSignature: &messages.RSASignature{Signature: requestSig},
	}

	// Request key with mock message
	_, err = RequestClientKey(instance, signedRequest, auth)
	if err == nil ||
		!strings.HasSuffix(err.Error(), gorsa.ErrVerification.Error()) {
		t.Errorf("Expected error case: " +
			"Registration signature should have failed.")
	}

}

// Error path: Construct a SignedClientKeyRequest message
// with bad marshal data in the ClientKeyRequest field.
func TestRequestClientKey_UnmarshalRequestError(t *testing.T) {
	instance, _, _, _, _, _, _, _ := setup(t)

	// Construct host for auth
	gwHost, err := connect.NewHost(&id.TempGateway, "", make([]byte, 0), connect.GetDefaultHostParams())
	if err != nil {
		t.Errorf("Unable to create gateway host: %+v", err)
	}

	// Construct auth
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          gwHost,
	}

	// Construct signed request with bad marshal data
	badMarshal := []byte("bad marshal data ")
	signedRequest := &pb.SignedClientKeyRequest{
		ClientKeyRequest: badMarshal,
	}

	// Request key with mock message
	_, err = RequestClientKey(instance, signedRequest, auth)
	if err == nil {
		t.Fatalf("Expected errror case: Should not be able to unmarshal ClientKeyRequest")
	}

}

// Error path: Construct a ClientRegistrationConfirmation message
// with bad marshal data in the ClientTransmissionConfirmation field.
func TestRequestClientKey_UnmarshalRegistrationConfirmError(t *testing.T) {
	instance, userRsaPub, userRsaPriv, _, userDhPubKey, clientRegistrarPrivKey, _, _ := setup(t)

	// Construct host for auth
	gwHost, err := connect.NewHost(&id.TempGateway, "", make([]byte, 0), connect.GetDefaultHostParams())
	if err != nil {
		t.Errorf("Unable to create gateway host: %+v", err)
	}

	// Construct auth
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          gwHost,
	}

	// Generate a pre-canned time for consistent testing
	testTime, err := time.Parse(time.RFC3339,
		"2012-12-21T22:08:41+00:00")
	if err != nil {
		t.Fatalf("RequestNonce error: "+
			"Could not parse precanned time: %v", err.Error())
	}
	// Convert public key to PEM
	clientRSAPubKeyPEM := rsa.CreatePublicKeyPem(userRsaPub)

	// Sign timestamp
	sigReg, err := registration.SignWithTimestamp(csprng.NewSystemRNG(),
		clientRegistrarPrivKey, testTime.UnixNano(), string(clientRSAPubKeyPEM))
	if err != nil {
		t.Errorf("Could not sign client's RSA key with registration's "+
			"key: %+v", err)
	}

	badMarshal := []byte("bad marshal data")

	salt := make([]byte, 32)
	copy(salt, "saltData")
	// Construct request
	request := &pb.ClientKeyRequest{
		Salt: salt,
		ClientTransmissionConfirmation: &pb.SignedRegistrationConfirmation{
			ClientRegistrationConfirmation: badMarshal,
			RegistrarSignature:             &messages.RSASignature{Signature: sigReg},
		},
		RegistrationTimestamp: testTime.UnixNano(),
		ClientDHPubKey:        userDhPubKey.Bytes(),
	}

	// Marshal request
	requestBytes, err := proto.Marshal(request)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	// Hash request
	opts := rsa.NewDefaultOptions()
	h := opts.Hash.New()
	h.Write(requestBytes)
	hashedData := h.Sum(nil)

	// Sign the request with the user's private key
	requestSig, err := rsa.Sign(csprng.NewSystemRNG(), userRsaPriv, opts.Hash, hashedData, opts)
	if err != nil {
		t.Fatalf("Sign error: %v", err)
	}

	// Construct signed request
	signedRequest := &pb.SignedClientKeyRequest{
		ClientKeyRequest:          requestBytes,
		ClientKeyRequestSignature: &messages.RSASignature{Signature: requestSig},
	}

	// Request key with mock message
	_, err = RequestClientKey(instance, signedRequest, auth)
	if err == nil {
		t.Fatalf("Expected errror case: " +
			"Should not be able to unmarshal ClientRegistrationConfirmation")
	}
}

// Error path: Attempt register when node secret manager is empty.
func TestRequestClientKey_NoNodeSecret(t *testing.T) {
	instance, userRsaPub, userRsaPriv, _, userDhPubKey, clientRegistrarPrivKey, _, _ := setup(t)

	// Construct host for auth
	gwHost, err := connect.NewHost(&id.TempGateway, "", make([]byte, 0), connect.GetDefaultHostParams())
	if err != nil {
		t.Errorf("Unable to create gateway host: %+v", err)
	}

	// Construct auth
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          gwHost,
	}

	// Generate a pre-canned time for consistent testing
	testTime, err := time.Parse(time.RFC3339,
		"2012-12-21T22:08:41+00:00")
	if err != nil {
		t.Fatalf("RequestNonce error: "+
			"Could not parse precanned time: %v", err.Error())
	}
	// Convert public key to PEM
	clientRSAPubKeyPEM := rsa.CreatePublicKeyPem(userRsaPub)

	// Sign timestamp
	sigReg, err := registration.SignWithTimestamp(csprng.NewSystemRNG(),
		clientRegistrarPrivKey, testTime.UnixNano(), string(clientRSAPubKeyPEM))
	if err != nil {
		t.Errorf("Could not sign client's RSA key with registration's "+
			"key: %+v", err)
	}

	regConfirm := &pb.ClientRegistrationConfirmation{
		RSAPubKey: string(clientRSAPubKeyPEM),
		Timestamp: testTime.UnixNano(),
	}

	regConfirmBytes, err := proto.Marshal(regConfirm)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	salt := make([]byte, 32)
	copy(salt, "saltData")
	// Construct request
	request := &pb.ClientKeyRequest{
		Salt: salt,
		ClientTransmissionConfirmation: &pb.SignedRegistrationConfirmation{
			ClientRegistrationConfirmation: regConfirmBytes,
			RegistrarSignature:             &messages.RSASignature{Signature: sigReg},
		},
		RegistrationTimestamp: testTime.UnixNano(),
		ClientDHPubKey:        userDhPubKey.Bytes(),
	}

	// Marshal request
	requestBytes, err := proto.Marshal(request)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	// Hash request
	opts := rsa.NewDefaultOptions()
	h := opts.Hash.New()
	h.Write(requestBytes)
	hashedData := h.Sum(nil)

	// Sign the request with the user's private key
	requestSig, err := rsa.Sign(csprng.NewSystemRNG(), userRsaPriv, opts.Hash, hashedData, opts)
	if err != nil {
		t.Fatalf("Sign error: %v", err)
	}

	// Construct signed request
	signedRequest := &pb.SignedClientKeyRequest{
		ClientKeyRequest:          requestBytes,
		ClientKeyRequestSignature: &messages.RSASignature{Signature: requestSig},
	}

	// Set up empty manager
	instance.SetSecretManagerTesting(t, storage.NewNodeSecretManager())

	// Request key with mock message
	_, err = RequestClientKey(instance, signedRequest, auth)
	if err == nil || !strings.ContainsAny(err.Error(), storage.NoSecretExistsError) {
		t.Fatalf("RequestClientKey error: %v", err)
	}

}

func setup(t interface{}) (*internal.Instance, *rsa.PublicKey, *rsa.PrivateKey, *cyclic.Int, *cyclic.Int, *rsa.PrivateKey, *id.ID, string) {
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
	nodeAddr := fmt.Sprintf("0.0.0.0:%d", 7000+rand.Intn(1000)+cnt)
	cnt++
	def := internal.Definition{
		ID:               nid,
		ResourceMonitor:  &measure.ResourceMonitor{},
		PrivateKey:       serverRSAPriv,
		PublicKey:        serverRSAPub,
		TlsCert:          cert,
		TlsKey:           key,
		FullNDF:          testUtil.NDF,
		PartialNDF:       testUtil.NDF,
		ListeningAddress: nodeAddr,
		DevMode:          true,
		RngStreamGen: fastRNG.NewStreamGenerator(10000,
			uint(runtime.NumCPU()), csprng.NewSystemRNG),
	}

	def.Network.PublicKey = regPKey.GetPublic()
	nodeIDs := make([]*id.ID, 0)
	nodeIDs = append(nodeIDs, nid)
	def.Gateway.ID = &id.TempGateway

	mach := state.NewTestMachine(dummyStates, current.PRECOMPUTING, t)

	instance, _ := internal.CreateServerInstance(&def, NewImplementation, mach, "1.1.0")

	return instance, cRsaPub, cRsaPriv, clintDHPriv, cDhPub, regPKey, nid, nodeAddr
}

func createMockInstance(t *testing.T, instIndex int, s current.Activity) (*internal.Instance, *connect.Circuit, *cyclic.Group) {
	grp := initImplGroup()
	nodeAddr := fmt.Sprintf("0.0.0.0:%d", 7000+rand.Intn(1000)+cnt)

	cnt++

	topology := connect.NewCircuit(BuildMockNodeIDs(5, t))
	def := internal.Definition{
		ResourceMonitor: &measure.ResourceMonitor{},
		FullNDF:         testUtil.NDF,
		PartialNDF:      testUtil.NDF,
		Flags:           internal.Flags{OverrideInternalIP: "0.0.0.0"},
		Gateway: internal.GW{
			ID: &id.TempGateway,
		},
		MetricsHandler: func(i *internal.Instance, roundID id.Round) error {
			return nil
		},
		ListeningAddress: nodeAddr,
		DevMode:          true,
		RngStreamGen: fastRNG.NewStreamGenerator(10000,
			uint(runtime.NumCPU()), csprng.NewSystemRNG),
	}

	privKey, err := rsa.GenerateKey(cryptoRand.Reader, 1024)
	if err != nil {
		t.Fatalf("Failed to generate priv key: %v", err)
	}
	def.PrivateKey = privKey
	def.ID = topology.GetNodeAtIndex(instIndex)

	m := state.NewTestMachine(dummyStates, s, t)

	instance, _ := internal.CreateServerInstance(&def, NewImplementation, m, "1.1.0")
	rnd, err := round.New(grp, id.Round(0), make([]phase.Phase, 0), make(phase.ResponseMap), topology, topology.GetNodeAtIndex(0), 3, instance.GetRngStreamGen(), nil, "0.0.0.0", nil, nil, instance.GetSecretManager(), nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}
	instance.GetRoundManager().AddRound(rnd)

	return instance, topology, grp
}

const primeString = "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
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
