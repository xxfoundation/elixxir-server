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
	"gitlab.com/elixxir/server/cmd/conf"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/services"
	"math/rand"
	"testing"
)

//TODO: make makeMsg it NOT random

// Fill part of message with random payload and associated data
func makeMsg() *format.Message {
	rng := rand.New(rand.NewSource(21))
	payloadA := make([]byte, f)
	payloadB := make([]byte, format.PayloadLen)
	rng.Read(payloadA)
	rng.Read(payloadB)
	msg := format.NewMessage()
	msg.SetPayloadA(payloadA)
	msg.SetDecryptedPayloadB(payloadB)

	return msg
}

func TestClientServer(t *testing.T) {
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

	mod := Keygen.DeepCopy()
	mod.Cryptop = cryptops.Keygen
	//guessing i dont need, reconfig for something in client

	nid := server.GenerateId()
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2), large.NewInt(2))

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
	// make a salt for testing
	testSalt := []byte("sodium chloride")
	// pad to length of the base key
	testSalt = append(testSalt, make([]byte, 256/8-len(testSalt))...)

	//TODO: figure out what tf to do with these two...
	var chunk services.Chunk
	var stream KeygenTestStream
	//var s *KeygenSubStream
	//stream.Link(grp,10, instance)

	/* stream doesn't play nice (the below only exist with keygenteststream
		// Necessary to avoid crashing
		stream.users[0] = id.ZeroID
		// Not necessary to avoid crashing
		stream.salts[0] = []byte{}

		grp.SetUint64(stream.KeysA.Get(uint32(0)), uint64(0))
		grp.SetUint64(stream.KeysB.Get(uint32(0)), uint64(1000+0))
		stream.salts[0] = testSalt
		stream.users[0] = usr.ID

		baseKeys := make([]*cyclic.Int, registry.CountUsers())
		for i := 0; i < registry.CountUsers(); i++ {
			baseKeys[i] = usr.BaseKey
		}
		err := mod.Adapt(stream, mod.Cryptop, chunk)
		if err != nil {
			t.Error(err.Error())
		}
		msg := makeMsg()
		encMsg := cmix.ClientEncrypt(grp, msg, testSalt, baseKeys)
		fmt.Println(encMsg)
		/*
			//here you would pull basekeys from mutliple users, but we shall shorthand it
			baseKeys := make([]*cyclic.Int, 10)
			baseKeys[0] = usr.BaseKey
			message := makeMsg()

			encMsg := cmix.ClientEncrypt(grp, message, testSalt, baseKeys)
			mod.Adapt(stream,mod.Cryptop,chunk)
			keyA := cmix.ClientKeyGen(grp, testSalt, baseKeys)
			hash, err := blake2b.New256(nil)
			if err != nil {
				t.Error("E2E Client Encrypt could not get blake2b Hash")
			}
			hash.Reset()
			hash.Write(testSalt)
			keyB := cmix.ClientKeyGen(grp, hash.Sum(nil), baseKeys)

			keyAInv := grp.Inverse(keyA, grp.NewInt(1))
			keyBInv := grp.Inverse(keyB, grp.NewInt(1))
			DecPayloadA := grp.Mul(keyAInv, grp.NewIntFromBytes(encMsg.GetPayloadA()), grp.NewInt(1))
			DecPayloadB := grp.Mul(keyBInv, grp.NewIntFromBytes(encMsg.GetPayloadB()), grp.NewInt(1))

			decMsg := format.NewMessage()
			decMsg.SetPayloadA(DecPayloadA.Bytes())
			decMsg.SetDecryptedPayloadB(DecPayloadB.LeftpadBytes(format.PayloadLen))

			ok := true
			for ok {
				chunk, ok = g.GetOutput()
				for i := chunk.Begin(); i < chunk.End(); i++ {
					resultA := stream.KeysA.Get(uint32(i))
					resultB := stream.KeysB.Get(uint32(i))
					resultABytes := resultA.Bytes()
					resultBBytes := resultB.Bytes()
				}
			}
	/**/
}
