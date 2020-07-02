///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package graphs

import (
	"bytes"
	"fmt"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cmix"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/measure"
	"gitlab.com/elixxir/server/internal/state"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/elixxir/server/testUtil"
	"golang.org/x/crypto/blake2b"
	"runtime"
	"testing"
)

// Give compile error unless KeygenSubStream meets keygenSubStreamInterface
var _ KeygenSubStreamInterface = &KeygenSubStream{}

// Example stream that includes a KeygenSubStream and can be put in a graph
type KeygenTestStream struct {
	KeygenSubStream
	// put other parts of the stream here if you have any
}

func (*KeygenTestStream) GetName() string {
	return "KeygenTestStream"
}

func (s *KeygenTestStream) Link(grp *cyclic.Group, batchSize uint32, source ...interface{}) {
	instance := source[0].(*internal.Instance)
	// You may have to create these elsewhere and pass them to
	// KeygenSubStream's Link so they can be populated in-place by the
	// CommStream for the graph
	s.KeygenSubStream.LinkStream(grp,
		instance.GetUserRegistry(),
		make([][]byte, batchSize),
		make([][][]byte, batchSize),
		make([]*id.ID, batchSize),
		grp.NewIntBuffer(batchSize, grp.NewInt(1)),
		grp.NewIntBuffer(batchSize, grp.NewInt(1)),
	)
}

func (s *KeygenTestStream) Input(index uint32,
	msg *mixmessages.Slot) error {
	return nil
}

func (s *KeygenTestStream) Output(index uint32) *mixmessages.Slot {
	return nil
}

/*no longer valid test
// Test that triggers error cases in the keygen cryptop adapter
func TestKeygenStreamAdapt_Errors(t *testing.T) {
	// First error: failing type assert for stream
	err := Keygen.Adapt(nil, MockKeygenOp, services.NewChunk(0, 1))
	if err == nil {
		t.Error("Failing the type assert for Adapt should have given an error")
	}

	// Second error: failing type assert for cryptop
	err = Keygen.Adapt(&KeygenTestStream{}, nil, services.NewChunk(0, 1))
	if err == nil {
		t.Error("Failing the type assert for Adapt should have given an error")
	}

	// Third error: Slot includes a user that's not in the registry
	// First, create a keygen stream
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
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16), large.NewInt(2), large.NewInt(1283))
	// Since the user registry has no users,
	// any user we pass into the stream will cause an error
	nid := server.GenerateId()

	smallprime := fmt.Sprintf("%x", 1283)
	generator := fmt.Sprintf("%x", 2)
	cmix := map[string]string{
		"prime":      primeString,
		"smallprime": smallprime,
		"generator":  generator,
	}

	params := conf.Params{
		Groups: conf.Groups{
			CMix: cmix,
		},
		Node: conf.Node{
			IdfPaths: []string{nid.String()},
		},
	}
	instance := server.CreateServerInstance(&params, &globals.UserMap{}, nil, nil)
	var stream KeygenTestStream
	stream.Link(grp, 1, instance)
	stream.users[0] = &id.ZeroUser
	stream.salts[0] = []byte("cesium chloride")
	err = Keygen.Adapt(&stream, MockKeygenOp, services.NewChunk(0, 1))
	if err == nil {
		t.Error("Passing a user ID that wasn't in the Database didn't result in an error")
	}
}*/

var MockKeygenOp cryptops.KeygenPrototype = func(grp *cyclic.Group, salt []byte, baseKey, key *cyclic.Int) {
	// returns the base key XOR'd with the salt
	// this is the easiest way to ensure both pieces of data are passed to the
	// op from the adapter
	x := baseKey.Bytes()
	for i := range x {
		x[i] = salt[i] ^ x[i]
	}
	grp.SetBytes(key, x)
}

// High-level test of the reception keygen adapter
// Also demonstrates how it can be part of a graph that could potentially also
// do other things
func TestKeygenStreamInGraph(t *testing.T) {
	instance := mockServerInstance(t)
	registry := instance.GetUserRegistry()
	grp := instance.GetConsensus().GetCmixGroup()
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

	var stream KeygenTestStream
	batchSize := uint32(2)

	// make a salt for testing
	testSalt := []byte("sodium chloride")
	// pad to length of the base key
	testSalt = append(testSalt, make([]byte, 256/8-len(testSalt))...)

	salthash, err := blake2b.New256(nil)

	if err != nil {
		t.Fatalf("Keygen: Test could not get blake2b hash: %s", err.Error())
	}

	salthash.Write(testSalt)

	testHashedSalt := salthash.Sum(nil)

	PanicHandler := func(g, m string, err error) {
		panic(fmt.Sprintf("Error in module %s of graph %s: %s", g, m, err.Error()))
	}

	cmixHash, err := hash.NewCMixHash()

	if err != nil {
		t.Errorf("Could not get a hash for kmacs: %+v", err)
	}

	kmac := cmix.GenerateKMAC(testSalt, u.BaseKey, cmixHash)

	gc := services.NewGraphGenerator(4, uint8(runtime.NumCPU()), 1, 1.0)

	// run the module in a graph
	g := gc.NewGraph("test", &stream)
	mod := Keygen.DeepCopy()
	mod.Cryptop = MockKeygenOp
	g.First(mod)
	g.Last(mod)
	//Keygen.NumThreads = 1
	g.Build(batchSize, PanicHandler)
	//rb := round.NewBuffer(grp, batchSize, batchSize)
	g.Link(grp, instance)
	// So, it's necessary to fill in the parts in the expanded batch with dummy
	// data to avoid crashing, or we need to exclude those parts in the cryptop
	for i := 0; i < int(g.GetExpandedBatchSize()); i++ {
		// Necessary to avoid crashing
		stream.users[i] = &id.ZeroUser
		// Not necessary to avoid crashing
		stream.salts[i] = []byte{}

		grp.SetUint64(stream.KeysA.Get(uint32(i)), uint64(i))
		grp.SetUint64(stream.KeysB.Get(uint32(i)), uint64(1000+i))
		stream.salts[i] = testSalt
		stream.users[i] = u.ID
		stream.kmacs[i] = [][]byte{kmac}
	}
	// Here's the actual data for the test

	g.Run()
	go g.Send(services.NewChunk(0, g.GetExpandedBatchSize()), nil)

	ok := true
	var chunk services.Chunk

	for ok {
		chunk, ok = g.GetOutput()
		for i := chunk.Begin(); i < chunk.End(); i++ {
			// inspect stream output: XORing the salt with the output should
			// return the original base key
			resultA := stream.KeysA.Get(uint32(i))
			resultB := stream.KeysB.Get(uint32(i))
			resultABytes := resultA.Bytes()
			resultBBytes := resultB.Bytes()
			// So, why is ResultBytes 256 bytes long,
			// while testSalt is 32 bytes long?
			// retrieve the original base key to prove that both data were passed to
			// the cryptop
			for j := range resultABytes {
				resultABytes[j] = resultABytes[j] ^ testSalt[j]
				resultBBytes[j] = resultBBytes[j] ^ testHashedSalt[j]
			}

			// Check result and base key. They should be equal
			if !bytes.Equal(resultABytes, u.BaseKey.Bytes()) {
				t.Error("Keygen: Base key and result key A weren't equal")
			}

			if !bytes.Equal(resultBBytes, u.BaseKey.Bytes()) {
				t.Error("Keygen: Base key and result key B weren't equal")
			}
		}
	}
}

// High-level test of the reception keygen adapter when the user is not registerd
// Also demonstrates how it can be part of a graph that could potentially also
// do other things
func TestKeygenStreamInGraphUnRegistered(t *testing.T) {
	instance := mockServerInstance(t)
	registry := instance.GetUserRegistry()
	grp := instance.GetConsensus().GetCmixGroup()
	u := registry.NewUser(grp)
	u.IsRegistered = false
	registry.UpsertUser(u)

	// Reception base key should be around 256 bits long,
	// depending on generation, to feed the 256-bit hash
	if u.BaseKey.BitLen() < 250 || u.BaseKey.BitLen() > 256 {
		t.Errorf("Base key has wrong number of bits. "+
			"Had %v bits in reception base key",
			u.BaseKey.BitLen())
	}

	var stream KeygenTestStream
	batchSize := uint32(1)

	// make a salt for testing
	testSalt := []byte("sodium chloride")
	// pad to length of the base key
	testSalt = append(testSalt, make([]byte, 256/8-len(testSalt))...)

	salthash, err := blake2b.New256(nil)

	if err != nil {
		t.Fatalf("Keygen: Test could not get blake2b hash: %s", err.Error())
	}

	salthash.Write(testSalt)

	cmixHash, err := hash.NewCMixHash()

	if err != nil {
		t.Errorf("Could not get a hash for kmacs: %+v", err)
	}

	kmac := cmix.GenerateKMAC(testSalt, u.BaseKey, cmixHash)

	PanicHandler := func(g, m string, err error) {
		panic(fmt.Sprintf("Error in module %s of graph %s: %s", g, m, err.Error()))
	}

	gc := services.NewGraphGenerator(4, uint8(runtime.NumCPU()), 1, 1.0)

	// run the module in a graph
	g := gc.NewGraph("test", &stream)
	mod := Keygen.DeepCopy()
	mod.Cryptop = MockKeygenOp
	g.First(mod)
	g.Last(mod)
	//Keygen.NumThreads = 1
	g.Build(batchSize, PanicHandler)
	//rb := round.NewBuffer(grp, batchSize, batchSize)
	g.Link(grp, instance)
	// So, it's necessary to fill in the parts in the expanded batch with dummy
	// data to avoid crashing, or we need to exclude those parts in the cryptop
	for i := 0; i < int(g.GetExpandedBatchSize()); i++ {
		// Necessary to avoid crashing
		stream.users[i] = &id.ZeroUser
		// Not necessary to avoid crashing
		stream.salts[i] = []byte{}

		grp.SetUint64(stream.KeysA.Get(uint32(i)), uint64(i))
		grp.SetUint64(stream.KeysB.Get(uint32(i)), uint64(1000+i))
		stream.salts[i] = testSalt
		stream.users[i] = u.ID
		stream.kmacs[i] = [][]byte{kmac}
	}
	// Here's the actual data for the test

	g.Run()
	go g.Send(services.NewChunk(0, g.GetExpandedBatchSize()), nil)

	ok := true
	var chunk services.Chunk

	one := stream.Grp.NewInt(1)

	for ok {
		chunk, ok = g.GetOutput()
		for i := chunk.Begin(); i < chunk.End(); i++ {
			// inspect stream output: XORing the salt with the output should
			// return the original base key
			resultA := stream.KeysA.Get(uint32(i))
			resultB := stream.KeysB.Get(uint32(i))

			// Check result and base key. They should be equal
			if resultA.Cmp(one) != 0 {
				t.Error("Keygen: Result key A not blanked when user is " +
					"unregistered")
			}

			if resultB.Cmp(one) != 0 {
				t.Error("Keygen: Result key B not blanked when user is " +
					"unregistered")
			}
		}
	}
}

// High-level test of the reception keygen adapter when the KMAC is invalid
// Also demonstrates how it can be part of a graph that could potentially also
// do other things
func TestKeygenStreamInGraph_InvalidKMAC(t *testing.T) {
	instance := mockServerInstance(t)
	registry := instance.GetUserRegistry()
	grp := instance.GetConsensus().GetCmixGroup()
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

	var stream KeygenTestStream
	batchSize := uint32(2)

	// make a salt for testing
	testSalt := []byte("sodium chloride")
	// pad to length of the base key
	testSalt = append(testSalt, make([]byte, 256/8-len(testSalt))...)

	salthash, err := blake2b.New256(nil)

	if err != nil {
		t.Fatalf("Keygen: Test could not get blake2b hash: %s", err.Error())
	}

	salthash.Write(testSalt)

	testHashedSalt := salthash.Sum(nil)

	PanicHandler := func(g, m string, err error) {
		panic(fmt.Sprintf("Error in module %s of graph %s: %s", g, m, err.Error()))
	}

	kmac := make([]byte, 32)

	gc := services.NewGraphGenerator(4, uint8(runtime.NumCPU()), 1, 1.0)

	// run the module in a graph
	g := gc.NewGraph("test", &stream)
	mod := Keygen.DeepCopy()
	mod.Cryptop = MockKeygenOp
	g.First(mod)
	g.Last(mod)
	//Keygen.NumThreads = 1
	g.Build(batchSize, PanicHandler)
	//rb := round.NewBuffer(grp, batchSize, batchSize)
	g.Link(grp, instance)
	// So, it's necessary to fill in the parts in the expanded batch with dummy
	// data to avoid crashing, or we need to exclude those parts in the cryptop
	for i := 0; i < int(g.GetExpandedBatchSize()); i++ {
		// Necessary to avoid crashing
		stream.users[i] = &id.ZeroUser
		// Not necessary to avoid crashing
		stream.salts[i] = []byte{}

		grp.SetUint64(stream.KeysA.Get(uint32(i)), uint64(i))
		grp.SetUint64(stream.KeysB.Get(uint32(i)), uint64(1000+i))
		stream.salts[i] = testSalt
		stream.users[i] = u.ID
		stream.kmacs[i] = [][]byte{kmac}
	}
	// Here's the actual data for the test

	g.Run()
	go g.Send(services.NewChunk(0, g.GetExpandedBatchSize()), nil)

	ok := true
	var chunk services.Chunk

	one := stream.Grp.NewInt(1)

	for ok {
		chunk, ok = g.GetOutput()
		for i := chunk.Begin(); i < chunk.End(); i++ {
			// inspect stream output: XORing the salt with the output should
			// return the original base key
			resultA := stream.KeysA.Get(uint32(i))
			resultB := stream.KeysB.Get(uint32(i))
			resultABytes := resultA.Bytes()
			resultBBytes := resultB.Bytes()
			// So, why is ResultBytes 256 bytes long,
			// while testSalt is 32 bytes long?
			// retrieve the original base key to prove that both data were passed to
			// the cryptop
			for j := range resultABytes {
				resultABytes[j] = resultABytes[j] ^ testSalt[j]
				resultBBytes[j] = resultBBytes[j] ^ testHashedSalt[j]
			}

			// Check result and base key. They should be equal
			if resultA.Cmp(one) != 0 {
				t.Error("Keygen: Result key A not blanked when kmacs " +
					"dont match")
			}

			if resultB.Cmp(one) != 0 {
				t.Error("Keygen: Result key B not blanked when kmacs " +
					"dont match")
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
