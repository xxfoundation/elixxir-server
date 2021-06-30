///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////
package graphs

/**/
import (
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/crypto/cmix"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/internal/state"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/elixxir/server/storage"
	"gitlab.com/elixxir/server/testUtil"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"golang.org/x/crypto/blake2b"
	"math/rand"
	"reflect"
	"testing"
)

//TODO: make makeMsg it NOT random

// Fill part of message with random payloads
// Fill part of message with random payloads
func makeMsg(grp *cyclic.Group) *format.Message {
	primeLegnth := len(grp.GetPBytes())
	rng := rand.New(rand.NewSource(21))
	payloadA := make([]byte, primeLegnth)
	payloadB := make([]byte, primeLegnth)
	rng.Read(payloadA)
	rng.Read(payloadB)
	msg := format.NewMessage(primeLegnth)
	msg.SetPayloadA(payloadA)
	msg.SetPayloadB(payloadB)

	return &msg
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
		large.NewInt(2))

	rid := id.Round(42)

	//Generate everything needed to make a user
	nid := internal.GenerateId(t)
	def := internal.Definition{
		ID:              nid,
		ResourceMonitor: &measure.ResourceMonitor{},
		PartialNDF:      testUtil.NDF,
		FullNDF:         testUtil.NDF,
		Flags:           internal.Flags{OverrideInternalIP: "0.0.0.0"},
		DevMode:         true,
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
		"1.1.0", make(chan chan struct{}))
	registry := instance.GetStorage()
	usr := &storage.Client{
		Id:           nil,
		DhKey:        grp.NewInt(5).Bytes(),
		IsRegistered: false,
	}
	_ = registry.UpsertClient(usr)

	//Generate the user's key
	var chunk services.Chunk
	var stream KeygenTestStream
	// make a salt for testing
	testSalt := []byte("sodium chloride")
	// pad to length of the base key
	testSalt = append(testSalt, make([]byte, 256/8-len(testSalt))...)
	testSalts := make([][]byte, 0)
	testSalts = append(testSalts, testSalt)
	//Generate an array of users for linking
	usrs := make([]*id.ID, 0)
	usrId, _ := usr.GetId()
	usrs = append(usrs, usrId)
	//generate an array of keys for linking
	keys := grp.NewIntBuffer(1, usr.GetDhKey(grp))
	kmacs := make([][][]byte, 1)
	reporter := round.NewClientFailureReport(nid)
	stream.LinkStream(grp, registry, testSalts, kmacs, usrs, keys, keys, reporter, 0, 32)
	err := Keygen.Adapt(&stream, cryptops.Keygen, chunk)
	if err != nil {
		t.Error(err)
	}
	//Create an array of basekeys to prepare for encryption
	userBaseKeys := make([]*cyclic.Int, 0)
	//FIXME: This is probably wrong
	userBaseKeys = append(userBaseKeys, usr.GetDhKey(grp))

	//Generate a mock message
	inputMsg := makeMsg(grp)

	//Encrypt the input message
	encryptedMsg := cmix.ClientEncrypt(grp, *inputMsg, testSalt, userBaseKeys, rid)

	//Generate an encrypted message using the keys manually, test output agains encryptedMsg above
	hash, err := blake2b.New256(nil)
	if err != nil {
		t.Error("E2E Client Encrypt could not get blake2b Hash")
	}

	hash.Reset()
	hash.Write(testSalt)

	keyA := cmix.ClientKeyGen(grp, testSalt, rid, userBaseKeys)
	keyB := cmix.ClientKeyGen(grp, hash.Sum(nil), rid, userBaseKeys)
	keyA_Inv := grp.Inverse(keyA, grp.NewInt(1))
	keyB_Inv := grp.Inverse(keyB, grp.NewInt(1))

	multPayloadA := grp.NewInt(1)
	multPayloadB := grp.NewInt(1)

	grp.Mul(keyA_Inv, grp.NewIntFromBytes(encryptedMsg.GetPayloadA()), multPayloadA)
	grp.Mul(keyB_Inv, grp.NewIntFromBytes(encryptedMsg.GetPayloadB()), multPayloadB)
	primeLength := len(grp.GetPBytes())
	testMsg := format.NewMessage(primeLength)

	testMsg.SetPayloadA(multPayloadA.Bytes())
	testMsg.SetPayloadB(multPayloadB.LeftpadBytes(uint64(primeLength)))

	//Compare the payloads of the 2 messages
	if !reflect.DeepEqual(testMsg.GetPayloadA(), inputMsg.GetPayloadA()) {
		t.Errorf("EncryptDecrypt("+
			") did not produce the correct payloadA\n\treceived: %d\n"+
			"\texpected: %d", encryptedMsg.GetPayloadA(), testMsg.GetPayloadA())
	}

	if !reflect.DeepEqual(testMsg.GetPayloadB(), inputMsg.GetPayloadB()) {
		t.Errorf("EncryptDecrypt("+
			") did not produce the correct payloadB\n\treceived: %d\n"+
			"\texpected: %d", inputMsg.GetPayloadB(), testMsg.GetPayloadB())
	}
}

// NewImplementation creates a new implementation of the server.
// When a function is added to comms, you'll need to point to it here.
func NewImplementation(instance *internal.Instance) *node.Implementation {

	impl := node.NewImplementation()

	return impl
}
