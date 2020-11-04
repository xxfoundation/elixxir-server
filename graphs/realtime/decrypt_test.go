///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package realtime

import (
	"fmt"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/crypto/cmix"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/fastRNG"
	hash2 "gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/crypto/large"
	gpumaths "gitlab.com/elixxir/gpumathsgo"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/graphs"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/internal/state"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/elixxir/server/testUtil"
	"gitlab.com/xx_network/primitives/id"
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
		t.Errorf("DecryptStream.GetName(), Expected %s, Received %s", expected, ds.GetName())
	}
}

// Test that DecryptStream.Link() Links correctly
func TestDecryptStream_Link(t *testing.T) {

	instance := mockServerInstance(t)
	grp := instance.GetConsensus().GetCmixGroup()

	stream := KeygenDecryptStream{}

	batchSize := uint32(100)

	registry := instance.GetUserRegistry()
	u := registry.NewUser(grp)
	registry.UpsertUser(u)

	roundBuffer := round.NewBuffer(grp, batchSize, batchSize)
	testReporter := round.NewClientFailureReport()
	var streamPool *gpumaths.StreamPool
	var rng *fastRNG.StreamGenerator
	stream.Link(grp, batchSize, roundBuffer, registry, rng, streamPool, testReporter)

	checkIntBuffer(stream.EcrPayloadA, batchSize, "EcrPayloadA", grp.NewInt(1), t)
	checkIntBuffer(stream.EcrPayloadB, batchSize, "EcrPayloadB", grp.NewInt(1), t)
	checkIntBuffer(stream.KeysPayloadA, batchSize, "KeysPayloadA", grp.NewInt(1), t)
	checkIntBuffer(stream.KeysPayloadB, batchSize, "KeysPayloadB", grp.NewInt(1), t)

	checkStreamIntBuffer(stream.Grp, stream.R, roundBuffer.R, "roundBuffer.R", t)
	checkStreamIntBuffer(stream.Grp, stream.U, roundBuffer.U, "roundBuffer.U", t)

	if uint32(len(stream.Users)) != batchSize {
		t.Errorf("dispatchStream.link(): user slice not created at correct length."+
			"Expected: %v, Received: %v", batchSize, len(stream.Users))
	}

	if uint32(len(stream.Salts)) != batchSize {
		t.Errorf("dispatchStream.link(): salts slice not created at correct length."+
			"Expected: %v, Received: %v", batchSize, len(stream.Salts))
	}

	for itr, u := range stream.Users {
		if !reflect.DeepEqual(u, &id.ID{}) {
			t.Errorf("dispatchStream.link(): user is at slot %v not initilized properly", itr)
		}
	}
}

// Tests Input's happy path
func TestDecryptStream_Input(t *testing.T) {

	instance := mockServerInstance(t)
	grp := instance.GetConsensus().GetCmixGroup()
	batchSize := uint32(100)

	stream := &KeygenDecryptStream{}

	registry := instance.GetUserRegistry()
	u := registry.NewUser(grp)
	registry.UpsertUser(u)

	roundBuffer := round.NewBuffer(grp, batchSize, batchSize)
	testReporter := round.NewClientFailureReport()
	var streamPool *gpumaths.StreamPool
	var rng *fastRNG.StreamGenerator

	stream.Link(grp, batchSize, roundBuffer, registry, rng, streamPool, testReporter)

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
			PayloadA: expected[0],
			PayloadB: expected[1],
			SenderID: id.NewIdFromBytes(expected[2], t).Bytes(),
			Salt:     expected[3],
		}

		err := stream.Input(b, msg)
		if err != nil {
			t.Errorf("DecryptStream.Input() errored on slot %v: %+v", b, err)
		}

		if !reflect.DeepEqual(stream.EcrPayloadA.Get(b).Bytes(), expected[0]) {
			t.Errorf("DecryptStream.Input() incorrectly stored EcrPayloadA "+
				"data at %v\n\texpected: %+v\n\treceived: %+v",
				b, expected[0], stream.EcrPayloadA.Get(b).Bytes())
		}

		if !reflect.DeepEqual(stream.EcrPayloadB.Get(b).Bytes(), expected[1]) {
			t.Errorf("DecryptStream.Input() incorrectly stored EcrPayloadB "+
				"data at %v\n\texpected: %+v\n\treceived: %+v",
				b, expected[1], stream.EcrPayloadB.Get(b).Bytes())
		}

	}

}

// Tests that the input errors correctly when the index is outside of the batch
func TestDecryptStream_Input_OutOfBatch(t *testing.T) {

	instance := mockServerInstance(t)
	grp := instance.GetConsensus().GetCmixGroup()

	batchSize := uint32(100)

	stream := &KeygenDecryptStream{}

	registry := instance.GetUserRegistry()
	u := registry.NewUser(grp)
	registry.UpsertUser(u)

	roundBuffer := round.NewBuffer(grp, batchSize, batchSize)
	testReporter := round.NewClientFailureReport()
	var streamPool *gpumaths.StreamPool
	var rng *fastRNG.StreamGenerator

	stream.Link(grp, batchSize, roundBuffer, registry, rng, streamPool, testReporter)

	msg := &mixmessages.Slot{
		PayloadA: []byte{0},
		PayloadB: []byte{0},
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

	instance := mockServerInstance(t)
	grp := instance.GetConsensus().GetCmixGroup()

	batchSize := uint32(100)

	stream := &KeygenDecryptStream{}

	registry := instance.GetUserRegistry()
	u := registry.NewUser(grp)
	registry.UpsertUser(u)

	roundBuffer := round.NewBuffer(grp, batchSize, batchSize)
	testReport := round.NewClientFailureReport()
	var streamPool *gpumaths.StreamPool
	var rng *fastRNG.StreamGenerator

	stream.Link(grp, batchSize, roundBuffer, registry, rng, streamPool, testReport)

	val := large.NewIntFromString(primeString, 16)
	val = val.Mul(val, val)
	msg := &mixmessages.Slot{
		PayloadA: val.Bytes(),
		PayloadB: val.Bytes(),
	}

	err := stream.Input(batchSize-10, msg)

	if err != services.ErrOutsideOfGroup {
		t.Errorf("DecryptStream.Input() did not return an error when out of group")
	}
}

//  Tests that Input errors correct when the user id is invalid
func TestDecryptStream_Input_NonExistantUser(t *testing.T) {

	instance := mockServerInstance(t)
	grp := instance.GetConsensus().GetCmixGroup()

	batchSize := uint32(100)

	stream := &KeygenDecryptStream{}

	registry := instance.GetUserRegistry()
	u := registry.NewUser(grp)
	registry.UpsertUser(u)

	roundBuffer := round.NewBuffer(grp, batchSize, batchSize)
	testReport := round.NewClientFailureReport()
	var streamPool *gpumaths.StreamPool
	var rng *fastRNG.StreamGenerator

	stream.Link(grp, batchSize, roundBuffer, registry, rng, streamPool, testReport)

	msg := &mixmessages.Slot{
		SenderID: []byte{1, 2},
		PayloadA: large.NewInt(3).Bytes(),
		PayloadB: large.NewInt(4).Bytes(),
	}

	err := stream.Input(batchSize-10, msg)

	if err != globals.ErrUserIDTooShort {
		t.Errorf("DecryptStream.Input() did not return an non existant user error when given non-existant user")
	}

	msg2 := &mixmessages.Slot{
		SenderID: id.NewIdFromUInt(0, id.User, t).Bytes(),
		PayloadA: large.NewInt(3).Bytes(),
		PayloadB: large.NewInt(4).Bytes(),
	}

	err2 := stream.Input(batchSize-10, msg2)

	if err2 == globals.ErrUserIDTooShort {
		t.Errorf("DecryptStream.Input() returned an existant user error when given non-existant user")
	}

}

//  Tests that Input errors correct when the salt is invalid
func TestDecryptStream_Input_SaltLength(t *testing.T) {

	instance := mockServerInstance(t)
	grp := instance.GetConsensus().GetCmixGroup()

	batchSize := uint32(100)

	stream := &KeygenDecryptStream{}

	registry := instance.GetUserRegistry()
	u := registry.NewUser(grp)
	registry.UpsertUser(u)

	roundBuffer := round.NewBuffer(grp, batchSize, batchSize)
	var streamPool *gpumaths.StreamPool
	var rng *fastRNG.StreamGenerator
	reporter := round.NewClientFailureReport()
	stream.Link(grp, batchSize, roundBuffer, registry, streamPool, rng, reporter)

	msg := &mixmessages.Slot{
		SenderID: id.NewIdFromUInt(0, id.User, t).Bytes(),
		Salt:     []byte{1, 2, 3},
		PayloadA: large.NewInt(3).Bytes(),
		PayloadB: large.NewInt(4).Bytes(),
	}

	err := stream.Input(batchSize-10, msg)

	if err != globals.ErrSaltIncorrectLength {
		t.Errorf("DecryptStream.Input() did not return a salt incorrect length error when given non-existant user")
	}

	msg2 := &mixmessages.Slot{
		SenderID: id.NewIdFromUInt(0, id.User, t).Bytes(),
		Salt:     make([]byte, 32),
		PayloadA: large.NewInt(3).Bytes(),
		PayloadB: large.NewInt(4).Bytes(),
	}

	err2 := stream.Input(batchSize-10, msg2)

	if err2 == globals.ErrSaltIncorrectLength {
		t.Errorf("DecryptStream.Input() returned a salt incorrect length error when given non-existant user")
	}
}

// Tests that the output function returns a valid cmixMessage
func TestDecryptStream_Output(t *testing.T) {

	instance := mockServerInstance(t)
	grp := instance.GetConsensus().GetCmixGroup()

	batchSize := uint32(100)

	stream := &KeygenDecryptStream{}

	registry := instance.GetUserRegistry()
	u := registry.NewUser(grp)
	registry.UpsertUser(u)

	roundBuffer := round.NewBuffer(grp, batchSize, batchSize)
	var streamPool *gpumaths.StreamPool
	var rng *fastRNG.StreamGenerator
	reporter := round.NewClientFailureReport()
	stream.Link(grp, batchSize, roundBuffer, registry, rng, streamPool, reporter)

	for b := uint32(0); b < batchSize; b++ {

		senderId := &id.ID{}
		salt := make([]byte, 32)

		expected := [][]byte{
			senderId.Bytes(),
			salt,
			{byte(b + 1), 0},
			{byte(b + 1), 1},
		}

		msg := &mixmessages.Slot{
			SenderID: expected[0],
			Salt:     expected[1],
			PayloadA: expected[2],
			PayloadB: expected[3],
		}

		err := stream.Input(b, msg)
		if err != nil {
			t.Errorf("DecryptStream.Output() errored on slot %v: %s", b, err.Error())
		}

		output := stream.Output(b)

		if !reflect.DeepEqual(output.SenderID, expected[0]) {
			t.Errorf("DecryptStream.Output() incorrect received SenderID data at %v: Expected: %v, Received: %v",
				b, expected[0], output.SenderID)
		}

		if !reflect.DeepEqual(output.Salt, expected[1]) {
			t.Errorf("DecryptStream.Output() incorrect received Salt data at %v: Expected: %v, Received: %v",
				b, expected[1], output.Salt)
		}

		if !reflect.DeepEqual(output.PayloadA, expected[2]) {
			t.Errorf("DecryptStream.Output() incorrect received PayloadA data at %v: Expected: %v, Received: %v",
				b, expected[2], output.PayloadA)
		}

		if !reflect.DeepEqual(output.PayloadB, expected[3]) {
			t.Errorf("DecryptStream.Output() incorrect received PayloadB data at %v: Expected: %v, Received: %v",
				b, expected[3], output.PayloadB)
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

	instance := mockServerInstance(t)
	grp := instance.GetConsensus().GetCmixGroup()
	registry := instance.GetUserRegistry()
	u := registry.NewUser(grp)
	u.IsRegistered = true
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

	gc := services.NewGraphGenerator(4, uint8(runtime.NumCPU()), 1, 1.0)

	//Initialize graph
	g := graphInit(gc)

	g.Build(batchSize, PanicHandler)

	// Build the roundBuffer
	roundBuffer := round.NewBuffer(grp, g.GetBatchSize(), g.GetExpandedBatchSize())

	// Fill the fields of the roundBuffer object for testing
	for i := uint32(0); i < g.GetExpandedBatchSize(); i++ {

		grp.Set(roundBuffer.R.Get(i), grp.NewInt(int64(2*i+1)))
		grp.Set(roundBuffer.S.Get(i), grp.NewInt(int64(3*i+1)))
		grp.Set(roundBuffer.PayloadBPrecomputation.Get(i), grp.NewInt(int64(1)))
		grp.Set(roundBuffer.PayloadAPrecomputation.Get(i), grp.NewInt(int64(1)))

	}

	g.Link(grp, roundBuffer, registry)

	stream := g.GetStream().(*KeygenDecryptStream)

	expectedPayloadA := grp.NewIntBuffer(batchSize, grp.NewInt(1))
	expectedPayloadB := grp.NewIntBuffer(batchSize, grp.NewInt(1))

	kmacHash, err := hash2.NewCMixHash()
	if err != nil {
		t.Errorf("Could not get hash for KMACing")
	}

	// So, it's necessary to fill in the parts in the expanded batch with dummy
	// data to avoid crashing, or we need to exclude those parts in the cryptop
	for i := 0; i < int(g.GetExpandedBatchSize()); i++ {
		// Necessary to avoid crashing
		stream.Users[i] = &id.ZeroUser
		// Not necessary to avoid crashing
		stream.Salts[i] = []byte{}

		grp.SetUint64(stream.EcrPayloadA.Get(uint32(i)), uint64(i+1))
		grp.SetUint64(stream.EcrPayloadB.Get(uint32(i)), uint64(1000+i))

		grp.SetUint64(expectedPayloadA.Get(uint32(i)), uint64(i+1))
		grp.SetUint64(expectedPayloadB.Get(uint32(i)), uint64(1000+i))

		stream.Salts[i] = testSalt
		stream.Users[i] = u.ID
		stream.KMACS[i] = [][]byte{cmix.GenerateKMAC(testSalt, u.BaseKey, kmacHash)}
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

			user, _ := registry.GetUser(stream.Users[i], grp)

			cryptops.Keygen(grp, stream.Salts[i], user.BaseKey, keyA)

			hash.Reset()
			hash.Write(stream.Salts[i])

			cryptops.Keygen(grp, hash.Sum(nil), user.BaseKey, keyB)

			// Verify expected KeyA matches actual KeyPayloadA
			if stream.KeysPayloadA.Get(i).Cmp(keyA) != 0 {
				t.Error(fmt.Sprintf("RealtimeDecrypt: Payload A Keys not equal on slot %v expected %v received %v",
					i, keyA.Text(16), stream.KeysPayloadA.Get(i).Text(16)))
			}

			// Verify expected KeyB matches actual KeyPayloadB
			if stream.KeysPayloadB.Get(i).Cmp(keyB) != 0 {
				t.Error(fmt.Sprintf("RealtimeDecrypt: Payload B Keys not equal on slot %v expected %v received %v",
					i, keyB.Text(16), stream.KeysPayloadB.Get(i).Text(16)))
			}

			cryptops.Mul3(grp, keyA, stream.R.Get(i), expectedPayloadA.Get(i))
			cryptops.Mul3(grp, keyB, stream.U.Get(i), expectedPayloadB.Get(i))

			// test that expectedPayloadA.Get(i) == stream.EcrPayloadA.Get(i)
			if stream.EcrPayloadA.Get(i).Cmp(expectedPayloadA.Get(i)) != 0 {
				t.Error(fmt.Sprintf("RealtimeDecrypt: Ecr PayloadA not equal on slot %v expected %v received %v",
					i, expectedPayloadA.Get(i).Text(16), stream.EcrPayloadA.Get(i).Text(16)))
			}

			// test that expectedPayloadB.Get(i) == stream.EcrPayloadB.Get(i)
			if stream.EcrPayloadB.Get(i).Cmp(expectedPayloadB.Get(i)) != 0 {
				t.Error(fmt.Sprintf("RealtimeDecrypt: Ecr PayloadB not equal on slot %v expected %v received %v",
					i, expectedPayloadB.Get(i).Text(16), stream.EcrPayloadB.Get(i).Text(16)))
			}
		}
	}
}

func mockServerInstance(i interface{}) *internal.Instance {

	nid := internal.GenerateId(i)
	def := internal.Definition{
		ID:              nid,
		ResourceMonitor: &measure.ResourceMonitor{},
		UserRegistry:    &globals.UserMap{},
		FullNDF:         testUtil.NDF,
		PartialNDF:      testUtil.NDF,
		Flags:           internal.Flags{DisableIpOverride: true},
	}
	def.Gateway.ID = nid.DeepCopy()
	def.Gateway.ID.SetType(id.Gateway)

	var stateChanges [current.NUM_STATES]state.Change
	stateChanges[current.NOT_STARTED] = func(from current.Activity) error {
		return nil
	}
	stateChanges[current.WAITING] = func(from current.Activity) error {
		return nil
	}
	stateChanges[current.PRECOMPUTING] = func(from current.Activity) error {
		return nil
	}
	stateChanges[current.STANDBY] = func(from current.Activity) error {
		return nil
	}
	stateChanges[current.REALTIME] = func(from current.Activity) error {
		return nil
	}
	stateChanges[current.COMPLETED] = func(from current.Activity) error {
		return nil
	}
	stateChanges[current.ERROR] = func(from current.Activity) error {
		return nil
	}
	stateChanges[current.CRASH] = func(from current.Activity) error {
		return nil
	}

	sm := state.NewMachine(stateChanges)

	instance, _ := internal.CreateServerInstance(&def, NewImplementation, sm,
		"1.1.0")

	return instance
}

// NewImplementation creates a new implementation of the server.
// When a function is added to comms, you'll need to point to it here.
func NewImplementation(instance *internal.Instance) *node.Implementation {

	impl := node.NewImplementation()

	return impl
}
