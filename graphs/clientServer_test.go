////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package graphs

/**/
import (
	"fmt"
	"gitlab.com/elixxir/crypto/cmix"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/services"
	"golang.org/x/crypto/blake2b"
	"math/rand"
	"reflect"
	"testing"
)

//TODO: make makeMsg it NOT random

// Fill part of message with random payload and associated data
// Fill part of message with random payload and associated data
func makeMsg() *format.Message {
	rng := rand.New(rand.NewSource(21))
	payloadA := make([]byte, format.PayloadLen)
	payloadB := make([]byte, format.PayloadLen)
	rng.Read(payloadA)
	rng.Read(payloadB)
	msg := format.NewMessage()
	msg.SetPayloadA(payloadA)
	msg.SetDecryptedPayloadB(payloadB)

	return msg
}

func TestClientServer(t *testing.T) {
	//Generate a group
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
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2), large.NewInt(2))

	//Generate everything needed to make a user
	nid := server.GenerateId()
	def := server.Definition{
		ID:              nid,
		CmixGroup:       grp,
		Topology:        circuit.New([]*id.Node{nid}),
		ResourceMonitor: &measure.ResourceMonitor{},
		UserRegistry:    &globals.UserMap{},
	}
	instance := server.CreateServerInstance(&def)
	registry := instance.GetUserRegistry()
	usr := registry.NewUser(grp)
	registry.UpsertUser(usr)
	fmt.Println(usr.BaseKey.Bytes())


	//Generate the user's key
	var chunk services.Chunk
	var stream KeygenTestStream
	// make a salt for testing
	testSalt := []byte("sodium chloride")
	// pad to length of the base key
	testSalt = append(testSalt, make([]byte, 256/8-len(testSalt))...)
	testSalts := make([][]byte,0)
	testSalts = append(testSalts, testSalt)
	//Generate an array of users for linking
	usrs := make([]*id.User, 0)
	usrs = append(usrs, usr.ID)
	//generate an array of keys for linking
	keys := grp.NewIntBuffer(1, usr.BaseKey)
	stream.LinkStream(grp, registry, testSalts, usrs, keys, keys)
	err := Keygen.Adapt(&stream,cryptops.Keygen,chunk)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(usr.BaseKey.Bytes())
	//Create an array of basekeys to prepare for encryption
	userBaseKeys := make([]*cyclic.Int, 0)
	userBaseKeys = append(userBaseKeys, usr.BaseKey)



	//Generate a mock message
	inputMsg := makeMsg()

	//Encrypt the input message
	encryptedMsg := cmix.ClientEncrypt(grp, inputMsg, testSalt, userBaseKeys)

	//Generate an encrypted message using the keys manually, test output agains encryptedMsg above
	hash, err := blake2b.New256(nil)
	if err != nil {
		t.Error("E2E Client Encrypt could not get blake2b Hash")
	}

	hash.Reset()
	hash.Write(testSalt)

	keyA := cmix.ClientKeyGen(grp, testSalt, userBaseKeys)
	keyB := cmix.ClientKeyGen(grp, hash.Sum(nil), userBaseKeys)
	payloadA := grp.NewIntFromBytes(inputMsg.GetPayloadA())
	payloadB := grp.NewIntFromBytes(inputMsg.GetPayloadBForEncryption())
	multPayloadA := grp.NewInt(1)
	multPayloadB := grp.NewInt(1)

	grp.Mul(keyA, payloadA, multPayloadA)
	grp.Mul(keyB, payloadB, multPayloadB)

	testMsg := format.NewMessage()

	testMsg.SetPayloadA(multPayloadA.Bytes())
	testMsg.SetPayloadB(multPayloadB.Bytes())

	//Compare the payloads of the 2 messages
	if !reflect.DeepEqual(testMsg.GetPayloadA(), encryptedMsg.GetPayloadA()) {
		t.Errorf("EncryptDecrypt("+
			") did not produce the correct payload\n\treceived: %d\n"+
			"\texpected: %d", encryptedMsg.GetPayloadA(), testMsg.GetPayloadA())
	}

	if !reflect.DeepEqual(testMsg.GetPayloadB(), encryptedMsg.GetPayloadB()) {
		t.Errorf("EncryptDecrypt("+
			") did not produce the correct associated data\n\treceived: %d\n"+
			"\texpected: %d", encryptedMsg.GetPayloadB(), testMsg.GetPayloadB())
	}
}
