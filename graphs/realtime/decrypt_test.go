////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package realtime

import (
	"fmt"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/graphs"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/measure"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/services"
	"golang.org/x/crypto/blake2b"
	"reflect"
	"runtime"
	"testing"
)

// Test that DecryptStream.GetName() returns the correct name
func TestDecryptStream_GetName(t *testing.T) {
	expected := "RealtimeDecryptStream"

	ds := KeygenDecryptStream{}

	if ds.GetName() != expected {
		t.Errorf("DecryptStream.GetName(), Expected %s, Recieved %s", expected, ds.GetName())
	}
}

// Test that DecryptStream.Link() Links correctly
func TestDecryptStream_Link(t *testing.T) {

	instance := mockServerInstance()
	grp := instance.GetGroup()

	stream := KeygenDecryptStream{}

	batchSize := uint32(100)

	registry := instance.GetUserRegistry()
	u := registry.NewUser(grp)
	registry.UpsertUser(u)

	roundBuffer := round.NewBuffer(grp, batchSize, batchSize)

	stream.Link(grp, batchSize, roundBuffer, registry)

	checkIntBuffer(stream.EcrMsg, batchSize, "EcrMsg", grp.NewInt(1), t)
	checkIntBuffer(stream.EcrAD, batchSize, "EcrAD", grp.NewInt(1), t)
	checkIntBuffer(stream.KeysMsg, batchSize, "KeysMsg", grp.NewInt(1), t)
	checkIntBuffer(stream.KeysAD, batchSize, "KeysAD", grp.NewInt(1), t)

	checkStreamIntBuffer(stream.Grp, stream.R, roundBuffer.R, "roundBuffer.R", t)
	checkStreamIntBuffer(stream.Grp, stream.U, roundBuffer.U, "roundBuffer.U", t)

	if uint32(len(stream.Users)) != batchSize {
		t.Errorf("dispatchStream.link(): user slice not created at correct length."+
			"Expected: %v, Recieved: %v", batchSize, len(stream.Users))
	}

	if uint32(len(stream.Salts)) != batchSize {
		t.Errorf("dispatchStream.link(): salts slice not created at correct length."+
			"Expected: %v, Recieved: %v", batchSize, len(stream.Salts))
	}

	for itr, u := range stream.Users {
		if !reflect.DeepEqual(u, &id.User{}) {
			t.Errorf("dispatchStream.link(): user is at slot %v not initilized properly", itr)
		}
	}
}

// Tests Input's happy path
func TestDecryptStream_Input(t *testing.T) {

	instance := mockServerInstance()
	grp := instance.GetGroup()
	batchSize := uint32(100)

	stream := &KeygenDecryptStream{}

	registry := instance.GetUserRegistry()
	u := registry.NewUser(grp)
	registry.UpsertUser(u)

	roundBuffer := round.NewBuffer(grp, batchSize, batchSize)

	stream.Link(grp, batchSize, roundBuffer, registry)

	for b := uint32(0); b < batchSize; b++ {

		expected := [][]byte{
			{byte(b + 1), 0},
			{byte(b + 1), 1},
			make([]byte, 32),
			make([]byte, 32),
		}

		expected[2][0] = byte(b + 1)
		expected[2][1] = byte(2)
		expected[3][0] = byte(b + 1)
		expected[3][1] = byte(3)

		msg := &mixmessages.Slot{
			MessagePayload: expected[0],
			AssociatedData: expected[1],
			SenderID:       expected[2],
			Salt:           expected[3],
		}

		err := stream.Input(b, msg)
		if err != nil {
			t.Errorf("DecryptStream.Input() errored on slot %v: %s", b, err.Error())
		}

		if !reflect.DeepEqual(stream.EcrMsg.Get(b).Bytes(), expected[0]) {
			t.Errorf("DecryptStream.Input() incorrect stored EcrMsg data at %v: Expected: %v, Recieved: %v",
				b, expected[0], stream.EcrMsg.Get(b).Bytes())
		}

		if !reflect.DeepEqual(stream.EcrAD.Get(b).Bytes(), expected[1]) {
			t.Errorf("DecryptStream.Input() incorrect stored EcrAD data at %v: Expected: %v, Recieved: %v",
				b, expected[1], stream.EcrAD.Get(b).Bytes())
		}

	}

}

// Tests that the input errors correctly when the index is outside of the batch
func TestDecryptStream_Input_OutOfBatch(t *testing.T) {

	instance := mockServerInstance()
	grp := instance.GetGroup()

	batchSize := uint32(100)

	stream := &KeygenDecryptStream{}

	registry := instance.GetUserRegistry()
	u := registry.NewUser(grp)
	registry.UpsertUser(u)

	roundBuffer := round.NewBuffer(grp, batchSize, batchSize)

	stream.Link(grp, batchSize, roundBuffer, registry)

	msg := &mixmessages.Slot{
		MessagePayload: []byte{0},
		AssociatedData: []byte{0},
	}

	err := stream.Input(batchSize, msg)

	if err != services.ErrOutsideOfBatch {
		t.Errorf("DecryptStream.Input() did not return an outside of batch error when out of batch")
	}

	err1 := stream.Input(batchSize+1, msg)

	if err1 != services.ErrOutsideOfBatch {
		t.Errorf("DecryptStream.Input() did not return an outside of batch error when out of batch")
	}
}

func TestSubAccess(t *testing.T) {
	decStream := &KeygenDecryptStream{}

	var stream services.Stream

	stream = decStream

	_, ok := stream.(graphs.KeygenSubStreamInterface)

	if !ok {
		t.Errorf("realtimeDecrypt: Could not access keygenStream")
	}

	_, ok2 := stream.(RealtimeDecryptSubStreamInterface)

	if !ok2 {
		t.Errorf("realtimeDecrypt: Could not access decryptStream")
	}
}

// Tests that Input errors correct when the passed value is out of the group
func TestDecryptStream_Input_OutOfGroup(t *testing.T) {
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

	instance := mockServerInstance()
	grp := instance.GetGroup()

	batchSize := uint32(100)

	stream := &KeygenDecryptStream{}

	registry := instance.GetUserRegistry()
	u := registry.NewUser(grp)
	registry.UpsertUser(u)

	roundBuffer := round.NewBuffer(grp, batchSize, batchSize)

	stream.Link(grp, batchSize, roundBuffer, registry)

	val := large.NewIntFromString(primeString, 16)
	val = val.Mul(val, val)
	msg := &mixmessages.Slot{
		MessagePayload: val.Bytes(),
		AssociatedData: val.Bytes(),
	}

	err := stream.Input(batchSize-10, msg)

	if err != services.ErrOutsideOfGroup {
		t.Errorf("DecryptStream.Input() did not return an error when out of group")
	}
}

//  Tests that Input errors correct when the user id is invalid
func TestDecryptStream_Input_NonExistantUser(t *testing.T) {

	instance := mockServerInstance()
	grp := instance.GetGroup()

	batchSize := uint32(100)

	stream := &KeygenDecryptStream{}

	registry := instance.GetUserRegistry()
	u := registry.NewUser(grp)
	registry.UpsertUser(u)

	roundBuffer := round.NewBuffer(grp, batchSize, batchSize)

	stream.Link(grp, batchSize, roundBuffer, registry)

	msg := &mixmessages.Slot{
		SenderID:       []byte{1, 2},
		MessagePayload: large.NewInt(3).Bytes(),
		AssociatedData: large.NewInt(4).Bytes(),
	}

	err := stream.Input(batchSize-10, msg)

	if err != globals.ErrUserIDTooShort {
		t.Errorf("DecryptStream.Input() did not return an non existant user error when given non-existant user")
	}

	msg2 := &mixmessages.Slot{
		SenderID:       id.NewUserFromUint(0, t).Bytes(),
		MessagePayload: large.NewInt(3).Bytes(),
		AssociatedData: large.NewInt(4).Bytes(),
	}

	err2 := stream.Input(batchSize-10, msg2)

	if err2 == globals.ErrUserIDTooShort {
		t.Errorf("DecryptStream.Input() returned an existant user error when given non-existant user")
	}

}

//  Tests that Input errors correct when the salt is invalid
func TestDecryptStream_Input_SaltLength(t *testing.T) {

	instance := mockServerInstance()
	grp := instance.GetGroup()

	batchSize := uint32(100)

	stream := &KeygenDecryptStream{}

	registry := instance.GetUserRegistry()
	u := registry.NewUser(grp)
	registry.UpsertUser(u)

	roundBuffer := round.NewBuffer(grp, batchSize, batchSize)

	stream.Link(grp, batchSize, roundBuffer, registry)

	msg := &mixmessages.Slot{
		SenderID:       id.NewUserFromUint(0, t).Bytes(),
		Salt:           []byte{1, 2, 3},
		MessagePayload: large.NewInt(3).Bytes(),
		AssociatedData: large.NewInt(4).Bytes(),
	}

	err := stream.Input(batchSize-10, msg)

	if err != globals.ErrSaltIncorrectLength {
		t.Errorf("DecryptStream.Input() did not return a salt incorrect length error when given non-existant user")
	}

	msg2 := &mixmessages.Slot{
		SenderID:       id.NewUserFromUint(0, t).Bytes(),
		Salt:           make([]byte, 32),
		MessagePayload: large.NewInt(3).Bytes(),
		AssociatedData: large.NewInt(4).Bytes(),
	}

	err2 := stream.Input(batchSize-10, msg2)

	if err2 == globals.ErrSaltIncorrectLength {
		t.Errorf("DecryptStream.Input() returned a salt incorrect length error when given non-existant user")
	}
}

// Tests that the output function returns a valid cmixMessage
func TestDecryptStream_Output(t *testing.T) {

	instance := mockServerInstance()
	grp := instance.GetGroup()

	batchSize := uint32(100)

	stream := &KeygenDecryptStream{}

	registry := instance.GetUserRegistry()
	u := registry.NewUser(grp)
	registry.UpsertUser(u)

	roundBuffer := round.NewBuffer(grp, batchSize, batchSize)

	stream.Link(grp, batchSize, roundBuffer, registry)

	for b := uint32(0); b < batchSize; b++ {

		senderId := &id.User{}
		salt := make([]byte, 32)

		expected := [][]byte{
			senderId.Bytes(),
			salt,
			{byte(b + 1), 0},
			{byte(b + 1), 1},
		}

		msg := &mixmessages.Slot{
			SenderID:       expected[0],
			Salt:           expected[1],
			MessagePayload: expected[2],
			AssociatedData: expected[3],
		}

		err := stream.Input(b, msg)
		if err != nil {
			t.Errorf("DecryptStream.Output() errored on slot %v: %s", b, err.Error())
		}

		output := stream.Output(b)

		if !reflect.DeepEqual(output.SenderID, expected[0]) {
			t.Errorf("DecryptStream.Output() incorrect recieved SenderID data at %v: Expected: %v, Recieved: %v",
				b, expected[0], output.SenderID)
		}

		if !reflect.DeepEqual(output.Salt, expected[1]) {
			t.Errorf("DecryptStream.Output() incorrect recieved Salt data at %v: Expected: %v, Recieved: %v",
				b, expected[1], output.Salt)
		}

		if !reflect.DeepEqual(output.MessagePayload, expected[2]) {
			t.Errorf("DecryptStream.Output() incorrect recieved MessagePayload data at %v: Expected: %v, Recieved: %v",
				b, expected[2], output.MessagePayload)
		}

		if !reflect.DeepEqual(output.AssociatedData, expected[3]) {
			t.Errorf("DecryptStream.Output() incorrect recieved AssociatedData data at %v: Expected: %v, Recieved: %v",
				b, expected[3], output.AssociatedData)
		}

	}

}

// Tests that DecryptStream conforms to the CommsStream interface
func TestDecryptStream_CommsInterface(t *testing.T) {

	var face interface{}
	face = &KeygenDecryptStream{}
	_, ok := face.(services.Stream)

	if !ok {
		t.Errorf("DecryptStream: Does not conform to the CommsStream interface")
	}

}

// High-level test of the reception keygen adapter
// Also demonstrates how it can be part of a graph that could potentially also
// do other things
func TestDecryptStreamInGraph(t *testing.T) {

	instance := mockServerInstance()
	grp := instance.GetGroup()
	registry := instance.GetUserRegistry()
	u := registry.NewUser(grp)
	registry.UpsertUser(u)

	// Reception base key should be around 256 bits long,
	// depending on generation, to feed the 256-bit hash
	if u.BaseKey.BitLen() < 250 || u.BaseKey.BitLen() > 256 {
		t.Errorf("Base key has wrong number of bits. "+
			"Had %v bits in reception base key",
			u.BaseKey.BitLen())
	}

	//var stream DecryptStream
	batchSize := uint32(32)
	//stream.Link(batchSize, &node.RoundBuffer{Grp: grp})

	// make a salt for testing
	testSalt := []byte("sodium chloride")
	// pad to length of the base key
	testSalt = append(testSalt, make([]byte, 256/8-len(testSalt))...)

	PanicHandler := func(g, m string, err error) {
		panic(fmt.Sprintf("Error in module %s of graph %s: %s", g, m, err.Error()))
	}

	// Show that the Init function meets the function type
	var graphInit graphs.Initializer
	graphInit = InitDecryptGraph

	gc := services.NewGraphGenerator(4, PanicHandler, uint8(runtime.NumCPU()), 1, 1.0)

	//Initialize graph
	g := graphInit(gc)

	g.Build(batchSize)

	// Build the roundBuffer
	roundBuffer := round.NewBuffer(grp, g.GetBatchSize(), g.GetExpandedBatchSize())

	// Fill the fields of the roundBuffer object for testing
	for i := uint32(0); i < g.GetExpandedBatchSize(); i++ {

		grp.Set(roundBuffer.R.Get(i), grp.NewInt(int64(2*i+1)))
		grp.Set(roundBuffer.S.Get(i), grp.NewInt(int64(3*i+1)))
		grp.Set(roundBuffer.ADPrecomputation.Get(i), grp.NewInt(int64(1)))
		grp.Set(roundBuffer.MessagePrecomputation.Get(i), grp.NewInt(int64(1)))

	}

	g.Link(grp, roundBuffer, registry)

	stream := g.GetStream().(*KeygenDecryptStream)

	expectedMsg := grp.NewIntBuffer(batchSize, grp.NewInt(1))
	expectedAD := grp.NewIntBuffer(batchSize, grp.NewInt(1))

	// So, it's necessary to fill in the parts in the expanded batch with dummy
	// data to avoid crashing, or we need to exclude those parts in the cryptop
	for i := 0; i < int(g.GetExpandedBatchSize()); i++ {
		// Necessary to avoid crashing
		stream.Users[i] = id.ZeroID
		// Not necessary to avoid crashing
		stream.Salts[i] = []byte{}

		grp.SetUint64(stream.EcrMsg.Get(uint32(i)), uint64(i+1))
		grp.SetUint64(stream.EcrAD.Get(uint32(i)), uint64(1000+i))

		grp.SetUint64(expectedMsg.Get(uint32(i)), uint64(i+1))
		grp.SetUint64(expectedAD.Get(uint32(i)), uint64(1000+i))

		stream.Salts[i] = testSalt
		stream.Users[i] = u.ID
	}
	// Here's the actual data for the test

	g.Run()
	go g.Send(services.NewChunk(0, g.GetExpandedBatchSize()), nil)

	ok := true
	var chunk services.Chunk
	hash, _ := blake2b.New256(nil)

	for ok {
		chunk, ok = g.GetOutput()

		for i := chunk.Begin(); i < chunk.End(); i++ {
			keyA := grp.NewInt(1)
			keyB := grp.NewInt(1)

			user, _ := registry.GetUser(stream.Users[i])

			cryptops.Keygen(grp, stream.Salts[i], user.BaseKey, keyA)

			hash.Reset()
			hash.Write(stream.Salts[i])

			cryptops.Keygen(grp, hash.Sum(nil), user.BaseKey, keyB)

			// Verify expected KeyA matches actual KeyMsg
			if stream.KeysMsg.Get(i).Cmp(keyA) != 0 {
				t.Error(fmt.Sprintf("RealtimeDecrypt: Message Keys not equal on slot %v expected %v received %v",
					i, keyA.Text(16), stream.KeysMsg.Get(i).Text(16)))
			}

			// Verify expected KeyB matches actual KeyAD
			if stream.KeysAD.Get(i).Cmp(keyB) != 0 {
				t.Error(fmt.Sprintf("RealtimeDecrypt: Message AD not equal on slot %v expected %v received %v",
					i, keyB.Text(16), stream.KeysAD.Get(i).Text(16)))
			}

			cryptops.Mul3(grp, keyA, stream.R.Get(i), expectedMsg.Get(i))
			cryptops.Mul3(grp, keyB, stream.U.Get(i), expectedAD.Get(i))

			// test that expectedMsg.Get(i) == stream.EcrMsg.Get(i)
			if stream.EcrMsg.Get(i).Cmp(expectedMsg.Get(i)) != 0 {
				t.Error(fmt.Sprintf("RealtimeDecrypt: Ecr message not equal on slot %v expected %v received %v",
					i, expectedMsg.Get(i).Text(16), stream.EcrMsg.Get(i).Text(16)))
			}

			// test that expectedAD.Get(i) == stream.EcrAD.Get(i)
			if stream.EcrAD.Get(i).Cmp(expectedAD.Get(i)) != 0 {
				t.Error(fmt.Sprintf("RealtimeDecrypt: Ecr AD not equal on slot %v expected %v received %v",
					i, expectedAD.Get(i).Text(16), stream.EcrAD.Get(i).Text(16)))
			}
		}
	}
}

func mockServerInstance() *server.Instance {
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

	def := server.Definition{
		ID:              nid,
		CmixGroup:       grp,
		Topology:        circuit.New([]*id.Node{nid}),
		ResourceMonitor: &measure.ResourceMonitor{},
		UserRegistry:    &globals.UserMap{},
	}

	instance := server.CreateServerInstance(&def)

	return instance
}
