///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package main

import (
	"fmt"
	"github.com/jinzhu/copier"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/graphs"
	"gitlab.com/elixxir/server/graphs/precomputation"
	"gitlab.com/elixxir/server/graphs/realtime"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"runtime"
	"sync"
	"testing"
)

const MODP768 = "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
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

const TinyStrongPrime = "6B" // 107

// ComputeSingleNodePrecomputation is a helper func to compute what
// the precomputation should be without any sharing computations for a
// single node system. In other words, it multiplies the R, S
// keys together for payload A's precomputation, and it does the same for
// the U, V keys to make payload B's precomputation.
func ComputeSingleNodePrecomputation(grp *cyclic.Group, round *round.Buffer) (
	*cyclic.Int, *cyclic.Int) {
	PayloadA := grp.NewInt(1)

	keys := round

	rInv := grp.NewMaxInt()
	sInv := grp.NewMaxInt()
	uInv := grp.NewMaxInt()
	vInv := grp.NewMaxInt()

	grp.Inverse(keys.R.Get(0), rInv)
	grp.Inverse(keys.S.Get(0), sInv)
	grp.Inverse(keys.U.Get(0), uInv)
	grp.Inverse(keys.V.Get(0), vInv)

	grp.Mul(PayloadA, rInv, PayloadA)
	grp.Mul(PayloadA, sInv, PayloadA)

	PayloadB := grp.NewInt(1)

	grp.Mul(PayloadB, uInv, PayloadB)
	grp.Mul(PayloadB, vInv, PayloadB)

	return PayloadA, PayloadB

}

// Compute Precomputation for N nodes
// NOTE: This does not handle precomputation under permutation, but it will
//       handle multi-node precomputation checks.
func ComputePrecomputation(grp *cyclic.Group, rounds []*round.Buffer,
	t *testing.T) (
	*cyclic.Int, *cyclic.Int) {
	PayloadA := grp.NewInt(1)
	PayloadB := grp.NewInt(1)
	rInv := grp.NewMaxInt()
	sInv := grp.NewMaxInt()
	uInv := grp.NewMaxInt()
	vInv := grp.NewMaxInt()

	for i, keys := range rounds {
		grp.Inverse(keys.R.Get(0), rInv)
		grp.Inverse(keys.S.Get(0), sInv)
		grp.Inverse(keys.U.Get(0), uInv)
		grp.Inverse(keys.V.Get(0), vInv)

		grp.Mul(PayloadA, rInv, PayloadA)
		grp.Mul(PayloadA, sInv, PayloadA)

		grp.Mul(PayloadB, uInv, PayloadB)
		grp.Mul(PayloadB, vInv, PayloadB)
		t.Logf("Node: %d, PayloadA: %v, PayloadB: %v\n", i,
			PayloadA.Bytes(), PayloadB.Bytes())
	}
	return PayloadA, PayloadB
}

// End to end test of the mathematical functions required to "share" 1
// key (i.e., R)
func RootingTest(grp *cyclic.Group, t *testing.T) {

	K1 := grp.NewInt(94)

	Z := grp.NewInt(11)

	Y1 := grp.NewInt(79)

	gZ := grp.NewInt(1)

	gY1 := grp.NewInt(1)

	MSG := grp.NewInt(1)
	CTXT := grp.NewInt(1)

	IVS := grp.NewInt(1)
	gY1c := grp.NewInt(1)

	RSLT := grp.NewInt(1)

	grp.Exp(grp.GetGCyclic(), Z, gZ)
	grp.RootCoprime(gZ, Z, RSLT)

	t.Logf("GENERATOR:\n\texpected: %#v\n\treceived: %#v\n",
		grp.GetGCyclic().Text(10), RSLT.Text(10))

	grp.Exp(grp.GetGCyclic(), Y1, gY1)
	grp.Mul(K1, gY1, MSG)

	grp.Exp(grp.GetGCyclic(), Z, gZ)
	grp.Exp(gZ, Y1, CTXT)

	grp.RootCoprime(CTXT, Z, gY1c)

	grp.Inverse(gY1c, IVS)

	grp.Mul(MSG, IVS, RSLT)

	t.Logf("ROOT TEST:\n\texpected: %#v\n\treceived: %#v",
		gY1.Text(10), gY1c.Text(10))

}

// End to end test of the mathematical functions required to "share" 2 keys
// (i.e., UV)
func RootingTestDouble(grp *cyclic.Group, t *testing.T) {

	K1 := grp.NewInt(94)
	K2 := grp.NewInt(18)

	Z := grp.NewInt(13)

	Y1 := grp.NewInt(87)
	Y2 := grp.NewInt(79)

	gZ := grp.NewInt(1)

	gY1 := grp.NewInt(1)
	gY2 := grp.NewInt(1)

	K2gY2 := grp.NewInt(1)

	gZY1 := grp.NewInt(1)
	gZY2 := grp.NewInt(1)

	K1gY1 := grp.NewInt(1)
	K1K2gY1Y2 := grp.NewInt(1)
	CTXT := grp.NewInt(1)

	IVS := grp.NewInt(1)
	gY1Y2c := grp.NewInt(1)

	RSLT := grp.NewInt(1)

	K1K2 := grp.NewInt(1)

	grp.Exp(grp.GetGCyclic(), Y1, gY1)
	grp.Mul(K1, gY1, K1gY1)

	grp.Exp(grp.GetGCyclic(), Y2, gY2)
	grp.Mul(K2, gY2, K2gY2)

	grp.Mul(K2gY2, K1gY1, K1K2gY1Y2)

	grp.Exp(grp.GetGCyclic(), Z, gZ)

	grp.Exp(gZ, Y1, gZY1)
	grp.Exp(gZ, Y2, gZY2)

	grp.Mul(gZY1, gZY2, CTXT)

	grp.RootCoprime(CTXT, Z, gY1Y2c)

	t.Logf("ROUND ASSOCIATED DATA PRIVATE KEY:\n\t%#v,\n", gY1Y2c.Text(10))

	grp.Inverse(gY1Y2c, IVS)

	grp.Mul(K1K2gY1Y2, IVS, RSLT)

	grp.Mul(K1, K2, K1K2)

	t.Logf("ROOT TEST DOUBLE:\n\texpected: %#v\n\treceived: %#v",
		RSLT.Text(10), K1K2.Text(10))

}

// End to end test of the mathematical functions required to "share" 3 keys
// (i.e., RST)
func RootingTestTriple(grp *cyclic.Group, t *testing.T) {

	K1 := grp.NewInt(26)
	K2 := grp.NewInt(77)
	K3 := grp.NewInt(100)

	Z := grp.NewInt(13)

	Y1 := grp.NewInt(69)
	Y2 := grp.NewInt(81)
	Y3 := grp.NewInt(13)

	gZ := grp.NewInt(1)

	gY1 := grp.NewInt(1)
	gY2 := grp.NewInt(1)
	gY3 := grp.NewInt(1)

	K1gY1 := grp.NewInt(1)
	K2gY2 := grp.NewInt(1)
	K3gY3 := grp.NewInt(1)

	gZY1 := grp.NewInt(1)
	gZY2 := grp.NewInt(1)
	gZY3 := grp.NewInt(1)

	gZY1Y2 := grp.NewInt(1)

	K1K2gY1Y2 := grp.NewInt(1)
	K1K2K3gY1Y2Y3 := grp.NewInt(1)

	CTXT := grp.NewInt(1)

	IVS := grp.NewInt(1)
	gY1Y2Y3c := grp.NewInt(1)

	RSLT := grp.NewInt(1)

	K1K2 := grp.NewInt(1)
	K1K2K3 := grp.NewInt(1)

	grp.Exp(grp.GetGCyclic(), Y1, gY1)
	grp.Mul(K1, gY1, K1gY1)

	grp.Exp(grp.GetGCyclic(), Y2, gY2)
	grp.Mul(K2, gY2, K2gY2)

	grp.Exp(grp.GetGCyclic(), Y3, gY3)
	grp.Mul(K3, gY3, K3gY3)

	grp.Mul(K2gY2, K1gY1, K1K2gY1Y2)
	grp.Mul(K1K2gY1Y2, K3gY3, K1K2K3gY1Y2Y3)

	grp.Exp(grp.GetGCyclic(), Z, gZ)

	grp.Exp(gZ, Y1, gZY1)
	grp.Exp(gZ, Y2, gZY2)
	grp.Exp(gZ, Y3, gZY3)

	grp.Mul(gZY1, gZY2, gZY1Y2)
	grp.Mul(gZY1Y2, gZY3, CTXT)

	grp.RootCoprime(CTXT, Z, gY1Y2Y3c)

	grp.Inverse(gY1Y2Y3c, IVS)

	grp.Mul(K1K2K3gY1Y2Y3, IVS, RSLT)

	grp.Mul(K1, K2, K1K2)
	grp.Mul(K1K2, K3, K1K2K3)

	t.Logf("ROOT TEST TRIPLE:\n\texpected: %#v\n\treceived: %#v",
		RSLT.Text(10), K1K2K3.Text(10))
}

// createDummyUserList creates a user list with a user of id 123,
// a base key of 1, and some random dsa params.
func createDummyUserList(grp *cyclic.Group,
	rng csprng.Source) *globals.UserMap {
	// Create a user -- FIXME: Why are we doing this here? Graphs shouldn't
	// need to be aware of users...it should be done and applied separately
	// as a list of keys to apply. This approach leads to getting part way
	// and then having time delays from user lookup and also sensitive
	// keying material being spewed all over in copies..
	registry := &globals.UserMap{}
	var userList []*globals.User
	u := new(globals.User)
	t := testing.T{}
	u.ID = id.NewIdFromUInt(uint64(123), id.User, &t)

	baseKeyBytes := []byte{1}
	u.BaseKey = grp.NewIntFromBytes(baseKeyBytes)
	// FIXME: This should really not be necessary and this API is wonky
	rsaPrivateKey, err := rsa.GenerateKey(csprng.NewSystemRNG(), 728)
	if err != nil {
		t.Errorf("Error getting RSA PK")
	}
	u.RsaPublicKey = rsaPrivateKey.GetPublic()
	registry.UpsertUser(u)
	userList = append(userList, u)
	return registry
}

func buildAndStartGraph(batchSize uint32, grp *cyclic.Group,
	roundBuf *round.Buffer, registry *globals.UserMap,
	rngStreamGen fastRNG.StreamGenerator, streams map[string]*DebugStream,
	t *testing.T) *services.Graph {
	//make the graph
	PanicHandler := func(g, m string, err error) {
		panic(fmt.Sprintf("Error in module %s of graph %s: %s",
			g, m, err.Error()))
	}
	// NOTE: input size greater than 1 would necessarily cause a hang here
	// since we never send more than 1 message through.
	gc := services.NewGraphGenerator(1,
		1, 1, 0)
	dGrph := InitDbgGraph(gc, streams, t, batchSize)
	dGrph.Build(batchSize, PanicHandler)

	dGrph.Link(grp, roundBuf, registry, &rngStreamGen)
	dGrph.Run()
	return dGrph
}

func buildAndStartGraph3(batchSize uint32, grp *cyclic.Group,
	roundBuf *round.Buffer, registry *globals.UserMap,
	rngStreamGen *fastRNG.StreamGenerator, streams map[string]*DebugStream,
	t *testing.T) *services.Graph {
	//make the graph
	PanicHandler := func(g, m string, err error) {
		panic(fmt.Sprintf("Error in module %s of graph %s: %s",
			g, m, err.Error()))
	}
	// NOTE: input size greater than 1 would necessarily cause a hang here
	// since we never send more than 1 message through.
	gc := services.NewGraphGenerator(1,
		1, 1, 0)
	dGrph := InitDbgGraph3(gc, streams, grp, batchSize, roundBuf, registry,
		rngStreamGen, t)
	dGrph.Build(batchSize, PanicHandler)

	dGrph.Link(grp, roundBuf, registry, rngStreamGen)
	dGrph.Run()
	return dGrph
}

// Perform an end to end test of the precomputation with batch size 1,
// then use it to send the message through a 1-node system to smoke test
// the cryptographic operations.
// NOTE: This test will not use real associated data, i.e., the recipientID val
// is not set in associated data.
// Trying to do this would lead to many changes:
// Firstly because the recipientID is place on bytes 2:33 of 256,
// meaning the second payload's representation in the group
// would be much bigger than the hardcoded P value of 107
// Secondly, the first byte of the second payload is randomly generated,
// so the expected values throughout the pipeline would need to be calculated
// Not having a proper second payload is not an issue in this particular test,
// because here only cryptops are chained
// The actual extraction of recipientID from associated data only occurs in
// handlers from the io package
func TestEndToEndCryptops(t *testing.T) {
	// Init, we use a small prime to make it easier to run the numbers
	// when debugging
	grp := cyclic.NewGroup(large.NewIntFromString(TinyStrongPrime, 16),
		large.NewInt(4))

	rngConstructor := NewPseudoRNG // FIXME: Why?
	rngStreamGen := fastRNG.NewStreamGenerator(10000,
		uint(runtime.NumCPU()), csprng.NewSystemRNG)
	batchSize := uint32(1)

	registry := createDummyUserList(grp, rngConstructor())
	dummyUser, _ := registry.GetUser(id.NewIdFromUInt(uint64(123), id.User, t), grp)

	//make the round buffer and manually set the round keys
	roundBuf := round.NewBuffer(grp, batchSize, batchSize)
	roundBuf.InitLastNode()
	grp.SetBytes(roundBuf.Z, []byte{13})
	grp.ExpG(roundBuf.Z, roundBuf.CypherPublicKey)
	grp.SetBytes(roundBuf.R.Get(0), []byte{26})
	grp.SetBytes(roundBuf.Y_R.Get(0), []byte{69})
	grp.SetBytes(roundBuf.S.Get(0), []byte{77})
	grp.SetBytes(roundBuf.Y_S.Get(0), []byte{81})
	grp.SetBytes(roundBuf.U.Get(0), []byte{94})
	grp.SetBytes(roundBuf.Y_U.Get(0), []byte{87})
	grp.SetBytes(roundBuf.V.Get(0), []byte{18})
	grp.SetBytes(roundBuf.Y_V.Get(0), []byte{79})

	streams := make(map[string]*DebugStream)

	dGrph := buildAndStartGraph(batchSize, grp, roundBuf, registry,
		*rngStreamGen, streams, t)
	megaStream := dGrph.GetStream().(*DebugStream)
	streams["END"] = megaStream

	// Create messsages
	megaStream.KeygenDecryptStream.Salts[0] = []byte{0}
	megaStream.KeygenDecryptStream.Users[0] = dummyUser.ID
	ecrPayloadA := grp.NewInt(31)
	ecrPayloadB := grp.NewInt(1)

	// Send message through the graph
	go func() {
		grp.Set(megaStream.DecryptStream.KeysPayloadA.Get(0), grp.NewInt(1))
		grp.Set(megaStream.DecryptStream.KeysPayloadB.Get(0), grp.NewInt(1))
		grp.Set(megaStream.DecryptStream.CypherPayloadA.Get(0),
			grp.NewInt(1))
		grp.Set(megaStream.DecryptStream.CypherPayloadB.Get(0), grp.NewInt(1))

		grp.Set(megaStream.KeygenDecryptStream.EcrPayloadA.Get(0), ecrPayloadA)
		grp.Set(megaStream.KeygenDecryptStream.EcrPayloadB.Get(0), ecrPayloadB)
		grp.Set(megaStream.KeygenDecryptStream.KeysPayloadA.Get(0),
			grp.NewInt(1))
		grp.Set(megaStream.KeygenDecryptStream.KeysPayloadB.Get(0),
			grp.NewInt(1))

		chunk := services.NewChunk(0, 1)
		dGrph.Send(chunk, nil)
	}()

	numDoneSlots := 0
	for chnk, ok := dGrph.GetOutput(); ok; chnk, ok =
		dGrph.GetOutput() {
		for i := chnk.Begin(); i < chnk.End(); i++ {
			numDoneSlots++
			t.Logf("done slot: %d, total done: %d",
				i, numDoneSlots)
		}
	}

	/* TODO: Check the following intermediate values
	From original/first version of code
	expectedDecrypt := []*cyclic.Int{
		grp.NewInt(5), grp.NewInt(17),
		grp.NewInt(79), grp.NewInt(36),
	}
	expectedPermute := []*cyclic.Int{
		grp.NewInt(23), grp.NewInt(61),
		grp.NewInt(39), grp.NewInt(85),
	}
	// Expected encrypt is deleted, we don't need it anymore!
	// The UV calcs are the same but the message parts aren't
	expectedReveal := []*cyclic.Int{
		grp.NewInt(42), grp.NewInt(13), // 42 -> 89 by manual calc!
	}
	expectedStrip := []*cyclic.Int{
		grp.NewInt(3), grp.NewInt(87),
	}
	 dGrph_test.go:552: DebugStream
	    dGrph_test.go:504: 1N Precomp Decrypt:
	            R([26], [69]), U([94], [87]),
	            KeysPayloadA/PayloadB: ([5] / [17]), CypherPayloadA/PayloadB: ([79] / [36])
	    dGrph_test.go:552: DebugStream
	    dGrph_test.go:513: 1N Precomp Permute: ([77], [81]),
		([18], [79]),
	             [23], [39], [61], [85]
	    dGrph_test.go:552: DebugStream
	    dGrph_test.go:513: 1N Precomp Permute: ([77], [81]),
		([18], [79]),
	             [23], [39], [61], [85]
	    dGrph_test.go:552: DebugStream
	    dGrph_test.go:528: 1N Precomp Reveal: [89], [13]
	    dGrph_test.go:552: DebugStream
	    dGrph_test.go:521: 1N Precomp Strip: [1], [1], [89], [13]
	    dGrph_test.go:552: DebugStream
	    dGrph_test.go:534: 1N RT Decrypt: K: [1], R: [26], M: [57],
		K: [1], U: [94], M: [94]
	    dGrph_test.go:552: DebugStream
	    dGrph_test.go:542: 1N RT Identify: S: [77], M: [2], V: [18],
		M: [87]
	    dGrph_test.go:398: done slot: 0, total done: 1
	    dGrph_test.go:428: PayloadA: 69 in GRP: xjz30UG9n4...,
		PayloadB: 16 in GRP: xjz30UG9n4...
	PayloadASS
	*/

	// These produce useful printouts when the test fails.
	RootingTest(grp, t)
	RootingTestDouble(grp, t)
	RootingTestTriple(grp, t)

	// Verify Precomputation
	/* TODO: We need to check for these intermediate values
	// NOTE: This is broken fornow until we understand why the deep copy
	// does'nt work
	ds := streams["Decrypt"].DecryptStream
	if ds.KeysPayloadA.Get(0).Cmp(grp.NewInt(5)) != 0 {
		t.Errorf("Precomp Decrypt KeysPayloadA: %v != [5]",
			ds.KeysPayloadA.Get(0).Bytes())
	}
	if ds.KeysPayloadB.Get(0).Cmp(grp.NewInt(17)) != 0 {
		t.Errorf("Precomp Decrypt KeysPayloadB: %v != [17]",
			ds.KeysPayloadB.Get(0).Bytes())
	}
	if ds.CypherPayloadA.Get(0).Cmp(grp.NewInt(79)) != 0 {
		t.Errorf("Precomp Decrypt CypherPayloadA: %v != [79]",
			ds.CypherPayloadA.Get(0).Bytes())
	}
	if ds.CypherPayloadB.Get(0).Cmp(grp.NewInt(36)) != 0 {
		t.Errorf("Precomp Decrypt CypherPayloadB: %v != [36]",
			ds.CypherPayloadB.Get(0).Bytes())
	}
	*/

	// Compute result directly
	PayloadA, PayloadB := ComputeSingleNodePrecomputation(grp, roundBuf)
	t.Logf("PayloadA: %s, PayloadB: %s",
		PayloadA.Text(10), PayloadB.Text(10))
	ss := streams["END"].StripStream
	if ss.PayloadAPrecomputation.Get(0).Cmp(PayloadA) != 0 {
		t.Errorf("%v != %v",
			ss.PayloadAPrecomputation.Get(0).Bytes(), PayloadA.Bytes())
	}
	if ss.PayloadBPrecomputation.Get(0).Cmp(PayloadB) != 0 {
		t.Errorf("%v != %v",
			ss.PayloadBPrecomputation.Get(0).Bytes(), PayloadB.Bytes())
	}

	/* Most of these are incorrect because we changed the computation to
	   2 keys instead of 3 as well as flipped to using inverse on clients
	expectedRTDecrypt := []*cyclic.Int{
		// 57 for PayloadA and 94 for PayloadB
		grp.NewInt(15), grp.NewInt(72),
	}
	expectedRTPermute := []*cyclic.Int{
		// 2
		grp.NewInt(13),
	}
	expectedRTIdentify := []*cyclic.Int{
		// 87
		grp.NewInt(61),
	}
	expectedRTPeel := []*cyclic.Int{
		// ???
		grp.NewInt(19),
	}
	*/

	// Verify Realtime

	expPayloadA := grp.NewInt(31)
	expPayloadB := grp.NewInt(1)
	is := streams["END"].IdentifyStream
	if is.EcrPayloadAPermuted[0].Cmp(expPayloadA) != 0 {
		t.Errorf("%v != %v", expPayloadA.Bytes(),
			megaStream.IdentifyStream.EcrPayloadAPermuted[0].Bytes())
	}
	if is.EcrPayloadBPermuted[0].Cmp(expPayloadB) != 0 {
		t.Errorf("%v != %v", expPayloadB.Bytes(),
			megaStream.IdentifyStream.EcrPayloadBPermuted[0].Bytes())
	}

}

// TestBatchSize3 runs the End to End test with 3 messages instead of 1
func TestBatchSize3(t *testing.T) {
	// Init, we use a small prime to make it easier to run the numbers
	// when debugging
	grp := cyclic.NewGroup(large.NewIntFromString(TinyStrongPrime, 16),
		large.NewInt(4))

	rngConstructor := NewPseudoRNG // FIXME: Why?
	rngStreamGen := fastRNG.NewStreamGenerator(10000,
		uint(runtime.NumCPU()), csprng.NewSystemRNG)
	batchSize := uint32(4)

	registry := createDummyUserList(grp, rngConstructor())
	dummyUser, _ := registry.GetUser(id.NewIdFromUInt(uint64(123), id.User, t), grp)

	//make the round buffer and manually set the round keys
	roundBuf := round.NewBuffer(grp, batchSize, batchSize)
	roundBuf.InitLastNode()
	grp.SetBytes(roundBuf.Z, []byte{13})
	grp.ExpG(roundBuf.Z, roundBuf.CypherPublicKey)
	for i := uint32(0); i < batchSize; i++ {
		grp.SetBytes(roundBuf.R.Get(i), []byte{26})
		grp.SetBytes(roundBuf.Y_R.Get(i), []byte{69})
		grp.SetBytes(roundBuf.S.Get(i), []byte{77})
		grp.SetBytes(roundBuf.Y_S.Get(i), []byte{81})
		grp.SetBytes(roundBuf.U.Get(i), []byte{94})
		grp.SetBytes(roundBuf.Y_U.Get(i), []byte{87})
		grp.SetBytes(roundBuf.V.Get(i), []byte{18})
		grp.SetBytes(roundBuf.Y_V.Get(i), []byte{79})
		roundBuf.Permutations[i] = (i + 1) % batchSize
	}
	// Compute result directly
	PayloadA, PayloadB := ComputeSingleNodePrecomputation(grp, roundBuf)

	streams := make(map[string]*DebugStream)

	dGrph := buildAndStartGraph(batchSize, grp, roundBuf, registry,
		*rngStreamGen, streams, t)
	megaStream := dGrph.GetStream().(*DebugStream)
	streams["END"] = megaStream

	// Create messsages
	for i := uint32(0); i < batchSize; i++ {
		megaStream.KeygenDecryptStream.Salts[i] = []byte{0}
		megaStream.KeygenDecryptStream.Users[i] = dummyUser.ID
	}

	// Send message through the graph
	for i := uint32(0); i < batchSize; i++ {
		ecrPayloadA := grp.NewInt((30+int64(i))%106 + 1)
		ecrPayloadB := grp.NewInt((int64(i))%106 + 1)

		grp.Set(megaStream.KeygenDecryptStream.EcrPayloadA.Get(i),
			ecrPayloadA)
		grp.Set(megaStream.KeygenDecryptStream.EcrPayloadB.Get(i),
			ecrPayloadB)
		grp.Set(megaStream.KeygenDecryptStream.KeysPayloadA.Get(i),
			grp.NewInt(1))
		grp.Set(megaStream.KeygenDecryptStream.KeysPayloadB.Get(i),
			grp.NewInt(1))

		grp.Set(megaStream.DecryptStream.KeysPayloadA.Get(i),
			grp.NewInt(1))
		grp.Set(megaStream.DecryptStream.KeysPayloadB.Get(i),
			grp.NewInt(1))
		grp.Set(megaStream.DecryptStream.CypherPayloadA.Get(i),
			grp.NewInt(1))
		grp.Set(megaStream.DecryptStream.CypherPayloadB.Get(i),
			grp.NewInt(1))
		chunk := services.NewChunk(i, i+1)
		dGrph.Send(chunk, nil)
	}

	numDoneSlots := 0
	for chunk, ok := dGrph.GetOutput(); ok; chunk, ok =
		dGrph.GetOutput() {
		for i := chunk.Begin(); i < chunk.End(); i++ {
			numDoneSlots++
		}
	}

	ss := streams["END"].StripStream
	is := streams["END"].IdentifyStream
	for i := uint32(0); i < batchSize; i++ {
		// Verify precomputation
		if ss.PayloadAPrecomputation.Get(i).Cmp(PayloadA) != 0 {
			t.Errorf("PRECOMPA %d: %v != %v", i,
				ss.PayloadAPrecomputation.Get(i).Bytes(),
				PayloadA.Bytes())
		}
		if ss.PayloadBPrecomputation.Get(i).Cmp(PayloadB) != 0 {
			t.Errorf("PRECOMPB %d: %v != %v", i,
				ss.PayloadBPrecomputation.Get(i).Bytes(),
				PayloadB.Bytes())
		}

		// Verify Realtime
		expPayloadA := grp.NewInt(int64(30 + (3+i)%batchSize + 1))
		expPayloadB := grp.NewInt(int64((3+i)%batchSize + 1))
		if is.EcrPayloadAPermuted[i].Cmp(expPayloadA) != 0 {
			t.Errorf("RTA %d: %v != %v", i,
				expPayloadA.Bytes(),
				is.EcrPayloadAPermuted[i].Bytes())
		}
		if is.EcrPayloadBPermuted[i].Cmp(expPayloadB) != 0 {
			t.Errorf("RTB %d: %v != %v", i,
				expPayloadB.Bytes(),
				is.EcrPayloadBPermuted[i].Bytes())
		}
	}
}

/* BEGIN TEST AND DUMMY STRUCTURES */

var DummyKeygen = services.Module{
	Adapt: func(s services.Stream, cryptop cryptops.Cryptop,
		chunk services.Chunk) error {
		streamInterface, ok := s.(graphs.KeygenSubStreamInterface)

		if !ok {
			return services.InvalidTypeAssert
		}

		kss := streamInterface.GetKeygenSubStream()

		for i := chunk.Begin(); i < chunk.End(); i++ {
			tmp := kss.Grp.NewInt(1)
			kss.Grp.Set(kss.KeysA.Get(i), tmp)

			kss.Grp.Set(kss.KeysB.Get(i), tmp)

		}

		return nil
	},
	Cryptop:    cryptops.Keygen,
	InputSize:  services.AutoInputSize,
	Name:       "DummyKeygen",
	NumThreads: services.AutoNumThreads,
}

// CreateStreamCopier takes a map object and copies the current state of the
// stream object to the map with the given key
func CreateStreamCopier(t *testing.T, key string,
	streams map[string]*DebugStream) *services.Module {
	return (&services.Module{
		Adapt: func(s services.Stream, cryptop cryptops.Cryptop,
			chunk services.Chunk) error {
			//ms := s.(*DebugStream)
			//streams[key] = ms.DeepCopy()
			//for i := chunk.Begin(); i < chunk.End(); i++ {
			//	fmt.Printf("%s: %d\n", key, i)
			//}
			return nil
		},
		Cryptop:        cryptops.Mul2,
		InputSize:      services.AutoInputSize,
		Name:           "DebugPrinter",
		NumThreads:     services.AutoNumThreads,
		StartThreshold: 0.0,
	}).DeepCopy()
}

type DebugStream struct {
	precomputation.GenerateStream
	precomputation.DecryptStream
	precomputation.PermuteStream
	precomputation.StripStream //Strip contains reveal
	realtime.KeygenDecryptStream
	realtime.IdentifyStream //Identify contains permute
	Outputs                 []*mixmessages.Slot
}

func (ds *DebugStream) GetName() string {
	return "DebugStream"
}

func (ds *DebugStream) DeepCopy() *DebugStream {
	ret := &DebugStream{}
	copier.Copy(ret, ds)
	return ret
}

func (ds *DebugStream) Link(grp *cyclic.Group, batchSize uint32,
	source ...interface{}) {
	roundBuf := source[0].(*round.Buffer)
	userRegistry := source[1].(*globals.UserMap)
	rngStreamGen := source[2].(*fastRNG.StreamGenerator)

	//Generate passthroughs for precomputation
	keysPayloadA := grp.NewIntBuffer(batchSize, grp.NewInt(1))
	cypherPayloadA := grp.NewIntBuffer(batchSize, grp.NewInt(1))
	keysPayloadB := grp.NewIntBuffer(batchSize, grp.NewInt(1))
	cypherPayloadB := grp.NewIntBuffer(batchSize, grp.NewInt(1))

	keysPayloadAPermuted := make([]*cyclic.Int, batchSize)
	cypherPayloadAPermuted := make([]*cyclic.Int, batchSize)
	keysPayloadBPermuted := make([]*cyclic.Int, batchSize)
	cypherPayloadBPermuted := make([]*cyclic.Int, batchSize)

	// Make sure we are using the same buffer for Msg precomps in roundbuf
	// FIXME: is there another way to do this?
	roundBuf.PermutedPayloadAKeys = keysPayloadAPermuted
	roundBuf.PermutedPayloadBKeys = keysPayloadBPermuted

	//Link precomputation
	ds.LinkGenerateStream(grp, batchSize, roundBuf, rngStreamGen)
	ds.LinkPrecompDecryptStream(grp, batchSize, roundBuf, nil, keysPayloadA,
		cypherPayloadA, keysPayloadB, cypherPayloadB)
	ds.LinkPrecompPermuteStream(grp, batchSize, roundBuf, nil, keysPayloadA,
		cypherPayloadA, keysPayloadB, cypherPayloadB, keysPayloadAPermuted, cypherPayloadAPermuted,
		keysPayloadBPermuted, cypherPayloadBPermuted)
	ds.LinkPrecompStripStream(grp, batchSize, roundBuf, nil, cypherPayloadA,
		cypherPayloadB)

	//Generate Passthroughs for realtime
	ecrPayloadA := grp.NewIntBuffer(batchSize, grp.NewInt(1))
	ecrPayloadB := grp.NewIntBuffer(batchSize, grp.NewInt(1))
	ecrPayloadAPermuted := make([]*cyclic.Int, batchSize)
	ecrPayloadBPermuted := make([]*cyclic.Int, batchSize)
	users := make([]*id.ID, batchSize)

	for i := uint32(0); i < batchSize; i++ {
		users[i] = &id.ID{}
	}

	ds.LinkRealtimeDecryptStream(grp, batchSize, roundBuf,
		userRegistry, ecrPayloadA, ecrPayloadB, grp.NewIntBuffer(batchSize,
			grp.NewInt(1)),
		grp.NewIntBuffer(batchSize, grp.NewInt(1)), users,
		make([][]byte, batchSize), make([][][]byte, batchSize))

	ds.LinkIdentifyStreams(grp, batchSize, roundBuf, ecrPayloadA, ecrPayloadB,
		ecrPayloadAPermuted, ecrPayloadBPermuted)
}

func (ds *DebugStream) Input(index uint32, slot *mixmessages.Slot) error {
	es := make([]error, 7)
	es[0] = ds.GenerateStream.Input(index, slot)
	es[1] = ds.DecryptStream.Input(index, slot)
	es[2] = ds.PermuteStream.Input(index, slot)
	es[3] = ds.StripStream.Input(index, slot)
	es[4] = ds.KeygenDecryptStream.Input(index, slot)
	es[5] = ds.IdentifyStream.Input(index, slot)
	es[6] = ds.StripStream.RevealStream.Input(index, slot)

	var lastErr error
	for i := 0; i < len(es); i++ {
		if es[i] != nil {
			// NOTE: Supressed because generally useless
			//fmt.Printf("Error DebugStream Input: %v\n", es[i])
			lastErr = es[i]
		}
	}
	return lastErr
}

func (ds *DebugStream) Output(index uint32) *mixmessages.Slot {
	ds.Outputs = make([]*mixmessages.Slot, 7)

	ds.Outputs[0] = ds.GenerateStream.Output(index)
	ds.Outputs[1] = ds.DecryptStream.Output(index)
	ds.Outputs[2] = ds.PermuteStream.Output(index)
	ds.Outputs[3] = ds.StripStream.Output(index)
	ds.Outputs[4] = ds.KeygenDecryptStream.Output(index)
	ds.Outputs[5] = ds.IdentifyStream.Output(index)
	ds.Outputs[6] = ds.StripStream.RevealStream.Output(index)

	for i := 0; i < len(ds.Outputs); i++ {
		ds.Outputs[i].Index = index
	}
	return nil
}

var ReintegratePrecompPermute = services.Module{
	Adapt: func(stream services.Stream, cryptop cryptops.Cryptop,
		chunk services.Chunk) error {
		mega, ok := stream.(*DebugStream)

		if !ok {
			return services.InvalidTypeAssert
		}

		ppsi := mega.PermuteStream

		for i := chunk.Begin(); i < chunk.End(); i++ {
			ppsi.Grp.Set(ppsi.KeysPayloadA.Get(i),
				ppsi.KeysPayloadAPermuted[i])
			ppsi.Grp.Set(ppsi.CypherPayloadA.Get(i),
				ppsi.CypherPayloadAPermuted[i])
			ppsi.Grp.Set(ppsi.KeysPayloadB.Get(i),
				ppsi.KeysPayloadBPermuted[i])
			ppsi.Grp.Set(ppsi.CypherPayloadB.Get(i),
				ppsi.CypherPayloadBPermuted[i])
		}
		return nil
	},
	Cryptop:        cryptops.Mul2,
	NumThreads:     1,
	InputSize:      1,
	Name:           "PrecompPermuteReintegration",
	StartThreshold: 1.0,
}

func InitDbgGraph(gc services.GraphGenerator, streams map[string]*DebugStream,
	t *testing.T, batchSize uint32) *services.Graph {
	g := gc.NewGraph("DbgGraph", &DebugStream{})

	//modules for precomputation
	//generate := precomputation.Generate.DeepCopy()
	decryptElgamal := precomputation.DecryptElgamal.DeepCopy()
	permuteElgamal := precomputation.PermuteElgamal.DeepCopy()
	permuteReintegrate := ReintegratePrecompPermute.DeepCopy()
	// NOTE: this corrects a race condition as this op cannot run
	// in parallel.
	permuteReintegrate.InputSize = batchSize
	revealRoot := precomputation.RevealRootCoprime.DeepCopy()
	stripInverse := precomputation.StripInverse.DeepCopy()
	stripMul2 := precomputation.StripMul2.DeepCopy()

	//modules for real time
	//decryptKeygen := DummyKeygen.DeepCopy()
	decryptMul3 := realtime.DecryptMul3.DeepCopy()
	permuteMul2 := realtime.PermuteMul2.DeepCopy()
	identifyMul2 := realtime.IdentifyMul2.DeepCopy()

	dPDecrypt := CreateStreamCopier(t, "Decrypt", streams)
	dPPermute := CreateStreamCopier(t, "Permute", streams)
	dPPermuteR := CreateStreamCopier(t, "Permute", streams)
	dPReveal := CreateStreamCopier(t, "Reveal", streams)
	dPStrip := CreateStreamCopier(t, "Strip", streams)
	dPStrip2 := CreateStreamCopier(t, "Strip", streams)

	dPDecryptRT := CreateStreamCopier(t, "DecryptRT", streams)
	dPPermuteMul2 := CreateStreamCopier(t, "PermuteRT", streams)

	//g.First(generate)
	// NOTE: Generate is skipped because it's values are hard coded
	//g.Connect(generate, decryptElgamal)
	g.First(decryptElgamal)
	g.Connect(decryptElgamal, dPDecrypt)
	g.Connect(dPDecrypt, permuteElgamal)
	g.Connect(permuteElgamal, dPPermute)
	g.Connect(dPPermute, permuteReintegrate)
	g.Connect(permuteReintegrate, dPPermuteR)
	g.Connect(dPPermuteR, revealRoot)
	g.Connect(revealRoot, dPReveal)
	g.Connect(dPReveal, stripInverse)
	g.Connect(stripInverse, dPStrip)
	g.Connect(dPStrip, stripMul2)
	g.Connect(stripMul2, dPStrip2)
	// NOTE: decryptKeyGen is skipped because it's values are hard coded
	g.Connect(dPStrip2, decryptMul3)
	g.Connect(decryptMul3, dPDecryptRT)
	g.Connect(dPDecryptRT, permuteMul2)
	g.Connect(permuteMul2, dPPermuteMul2)
	g.Connect(dPPermuteMul2, identifyMul2)
	g.Last(identifyMul2)
	return g
}

func wrapAdapt(batchSize uint32, outIdx int, name string, dStrms []*DebugStream,
	adaptFnc func(s services.Stream, cryptop cryptops.Cryptop,
		chunk services.Chunk) error, t *testing.T) func(s services.Stream,
	cryptop cryptops.Cryptop,
	chunk services.Chunk) error {
	adapt := func(s services.Stream, cryptop cryptops.Cryptop,
		chunk services.Chunk) error {
		var err error
		for i := uint32(0); i < batchSize; i++ {
			t.Logf("Running %s %d", name, i)
			err = adaptFnc(dStrms[i], cryptop, chunk)
			for j := chunk.Begin(); j < chunk.End(); j++ {
				// Copy output of current stream to tmp
				dStrms[i].Output(j)
				output := dStrms[i].Outputs[outIdx]
				// Input cur stream to next stream
				dStrms[(i+1)%3].Input(j, output)
				/*if err!=nil{
					t.Errorf("Errored on wrap adapt %v", err)
				}*/
				// Todo: Call link to copy right keys??
			}
		}
		return err
	}
	return adapt
}

func InitDbgGraph3(gc services.GraphGenerator, streams map[string]*DebugStream,
	grp *cyclic.Group, batchSize uint32, roundBuf *round.Buffer,
	registry *globals.UserMap, rngStreamGen *fastRNG.StreamGenerator,
	t *testing.T) *services.Graph {
	dStrms := make([]*DebugStream, 3)
	for i := 1; i < 3; i++ {
		dStrms[i] = &DebugStream{}
		dStrms[i].Link(grp, batchSize, roundBuf, registry, rngStreamGen)
	}
	dStrms[0] = &DebugStream{}
	g := gc.NewGraph("DbgGraph", dStrms[0])

	//modules for precomputation
	//generate := precomputation.Generate.DeepCopy()
	decryptElgamal := precomputation.DecryptElgamal.DeepCopy()
	decryptElgamal.Adapt = wrapAdapt(3, 1, "Decrypt", dStrms,
		decryptElgamal.Adapt, t)
	permuteElgamal := precomputation.PermuteElgamal.DeepCopy()
	permuteElgamal.Adapt = wrapAdapt(3, 2, "Permute", dStrms,
		permuteElgamal.Adapt, t)
	permuteReintegrate := ReintegratePrecompPermute.DeepCopy()
	// Note this prevents a race condition as this op cannot run in parallel
	permuteReintegrate.InputSize = batchSize
	revealRoot := precomputation.RevealRootCoprime.DeepCopy()
	revealRoot.Adapt = wrapAdapt(3, 6, "Reveal", dStrms,
		revealRoot.Adapt, t)
	stripInverse := precomputation.StripInverse.DeepCopy()
	stripMul2 := precomputation.StripMul2.DeepCopy()

	//modules for real time
	//decryptKeygen := DummyKeygen.DeepCopy()
	decryptMul3 := realtime.DecryptMul3.DeepCopy()
	decryptMul3.Adapt = wrapAdapt(3, 4, "DecryptRT", dStrms,
		decryptMul3.Adapt, t)

	permuteMul2 := realtime.PermuteMul2.DeepCopy()
	permuteMul2.Adapt = wrapAdapt(3, 5, "PermuteRT", dStrms,
		permuteMul2.Adapt, t)
	identifyMul2 := realtime.IdentifyMul2.DeepCopy()

	dPDecrypt := CreateStreamCopier(t, "Decrypt", streams)
	dPPermute := CreateStreamCopier(t, "Permute", streams)
	dPPermuteR := CreateStreamCopier(t, "Permute", streams)
	dPReveal := CreateStreamCopier(t, "Reveal", streams)
	dPStrip := CreateStreamCopier(t, "Strip", streams)
	dPStrip2 := CreateStreamCopier(t, "Strip", streams)

	dPDecryptRT := CreateStreamCopier(t, "DecryptRT", streams)
	dPPermuteMul2 := CreateStreamCopier(t, "PermuteRT", streams)

	//g.First(generate)
	// NOTE: Generate is skipped because it's values are hard coded
	//g.Connect(generate, decryptElgamal)
	g.First(decryptElgamal)
	g.Connect(decryptElgamal, dPDecrypt)
	g.Connect(dPDecrypt, permuteElgamal)
	g.Connect(permuteElgamal, dPPermute)
	g.Connect(dPPermute, permuteReintegrate)
	g.Connect(permuteReintegrate, dPPermuteR)
	g.Connect(dPPermuteR, revealRoot)
	g.Connect(revealRoot, dPReveal)
	g.Connect(dPReveal, stripInverse)
	g.Connect(stripInverse, dPStrip)
	g.Connect(dPStrip, stripMul2)
	g.Connect(stripMul2, dPStrip2)
	// NOTE: decryptKeyGen is skipped because it's values are hard coded
	g.Connect(dPStrip2, decryptMul3)
	g.Connect(decryptMul3, dPDecryptRT)
	g.Connect(dPDecryptRT, permuteMul2)
	g.Connect(permuteMul2, dPPermuteMul2)
	g.Connect(dPPermuteMul2, identifyMul2)
	g.Last(identifyMul2)
	return g
}

func RunDbgGraph(batchSize uint32, rngConstructor func() csprng.Source,
	t *testing.T) {
	grp := cyclic.NewGroup(large.NewIntFromString(MODP768, 16),
		large.NewInt(2))

	//nid := server.GenerateId()

	//instance := server.CreateServerInstance(grp, nid, &globals.UserMap{})

	registry := &globals.UserMap{}
	var userList []*globals.User

	var salts [][]byte

	rng := rngConstructor()

	//make the user IDs and their base keys and the salts
	for i := uint32(0); i < batchSize; i++ {
		u := registry.NewUser(grp)
		u.ID = id.NewIdFromUInt(uint64(i), id.User, t)

		u.BaseKey = grp.NewInt(1)
		registry.UpsertUser(u)

		salt := make([]byte, 32)
		salts = append(salts, salt)

		userList = append(userList, u)
	}

	var messageList []*cyclic.Int
	var PayloadBList []*cyclic.Int

	//make the messages
	for i := uint32(0); i < batchSize; i++ {
		messageBytes := make([]byte, 32)
		_, err := rng.Read(messageBytes)
		if err != nil {
			t.Error("DbgGraph: could not rng")
		}
		messageBytes[len(messageBytes)-1] |= 0x01
		messageList = append(messageList,
			grp.NewIntFromBytes(messageBytes))

		adBytes := make([]byte, 32)
		_, err = rng.Read(adBytes)
		if err != nil {
			t.Error("DbgGraph: could not rng")
		}
		adBytes[len(adBytes)-1] |= 0x01
		PayloadBList = append(PayloadBList, grp.NewIntFromBytes(adBytes))
	}

	var ecrPayloadAs []*cyclic.Int
	var ecrPayloadB []*cyclic.Int

	//encrypt the messages
	for i := uint32(0); i < batchSize; i++ {
		ecrPayloadAs = append(ecrPayloadAs, messageList[i])
		ecrPayloadB = append(ecrPayloadB, PayloadBList[i])
	}

	//make the graph
	PanicHandler := func(g, m string, err error) {
		panic(fmt.Sprintf("Error in module %s of graph %s: %s",
			g, m, err.Error()))
	}

	gc := services.NewGraphGenerator(1,
		uint8(runtime.NumCPU()), services.AutoOutputSize, 0)
	streams := make(map[string]*DebugStream)
	dGrph := InitDbgGraph(gc, streams, t, batchSize)

	dGrph.Build(batchSize, PanicHandler)

	//make the round buffer
	roundBuf := round.NewBuffer(grp, batchSize,
		dGrph.GetExpandedBatchSize())
	roundBuf.InitLastNode()

	//do a mock share phase
	zBytes := make([]byte, 31)
	rng.Read(zBytes)
	zBytes[0] |= 0x01
	zBytes[len(zBytes)-1] |= 0x01

	grp.SetBytes(roundBuf.Z, zBytes)
	grp.ExpG(roundBuf.Z, roundBuf.CypherPublicKey)

	dGrph.Link(grp, roundBuf, registry,
		fastRNG.NewStreamGenerator(10000, uint(runtime.NumCPU()),
			csprng.NewSystemRNG))

	stream := dGrph.GetStream()

	megaStream := stream.(*DebugStream)

	dGrph.Run()

	go func() {
		t.Log("Beginning test")
		for i := uint32(0); i < batchSize; i++ {
			megaStream.KeygenDecryptStream.Salts[i] = salts[i]
			megaStream.KeygenDecryptStream.Users[i] = userList[i].ID
			grp.Set(megaStream.IdentifyStream.EcrPayloadA.Get(i),
				ecrPayloadAs[i])
			grp.Set(megaStream.IdentifyStream.EcrPayloadB.Get(i),
				ecrPayloadB[i])
			chunk := services.NewChunk(i, i+1)
			dGrph.Send(chunk, nil)
		}
	}()

	numDoneSlots := 0

	for chunk, ok := dGrph.GetOutput(); ok; chunk, ok =
		dGrph.GetOutput() {
		for i := chunk.Begin(); i < chunk.End(); i++ {
			numDoneSlots++
			fmt.Println("done slot:", i, " total done:",
				numDoneSlots)
		}
	}

	for i := uint32(0); i < batchSize; i++ {
		if megaStream.IdentifyStream.EcrPayloadA.Get(i).Cmp(
			messageList[i]) != 0 {
			t.Errorf("DbgGraph: Decrypted message not"+
				" the same as message on slot %v,"+
				"Sent: %s, Decrypted: %s", i,
				messageList[i].Text(16),
				megaStream.IdentifyStream.EcrPayloadA.Get(
					i).Text(16))
		}
		if megaStream.IdentifyStream.EcrPayloadB.Get(i).Cmp(PayloadBList[i]) != 0 {
			t.Errorf("DbgGraph: Decrypted PayloadB not the same"+
				" as send message on slot %v, "+
				"Sent: %s, Decrypted: %s", i,
				PayloadBList[i].Text(16),
				megaStream.IdentifyStream.EcrPayloadB.Get(i).Text(16))
		}
	}

}

func Test_DebugStream(t *testing.T) {
	grp := cyclic.NewGroup(large.NewIntFromString(MODP768, 16),
		large.NewInt(2))

	batchSize := uint32(1000)

	PanicHandler := func(g, m string, err error) {
		panic(fmt.Sprintf("Error in module %s of graph %s: %s",
			g, m, err.Error()))
	}

	gc := services.NewGraphGenerator(1,
		uint8(runtime.NumCPU()), services.AutoOutputSize, 0)
	streams := make(map[string]*DebugStream)
	dGrph := InitDbgGraph(gc, streams, t, batchSize)

	dGrph.Build(batchSize, PanicHandler)

	//make the round buffer
	roundBuf := round.NewBuffer(grp, batchSize,
		dGrph.GetExpandedBatchSize())

	dGrph.Link(grp, roundBuf, &globals.UserMap{},
		fastRNG.NewStreamGenerator(10000, uint(runtime.NumCPU()),
			csprng.NewSystemRNG))

	stream := dGrph.GetStream()

	_, ok := stream.(precomputation.GenerateSubstreamInterface)

	if !ok {
		t.Errorf("DebugStream: type assert failed when " +
			"getting 'GenerateSubstreamInterface'")
	}

	_, ok = stream.(precomputation.PrecompDecryptSubstreamInterface)

	if !ok {
		t.Errorf("DebugStream: type assert failed when " +
			"getting 'PrecompDecryptSubstreamInterface'")
	}

}

func NewPseudoRNG() csprng.Source {
	return &PseudoRNG{
		r: rand.New(rand.NewSource(42)),
	}
}

type PseudoRNG struct {
	r *rand.Rand
	sync.Mutex
}

// Read calls the crypto/rand Read function and returns the values
func (p *PseudoRNG) Read(b []byte) (int, error) {
	p.Lock()
	defer p.Unlock()
	return p.r.Read(b)
}

// SetSeed has not effect on the system reader
func (p *PseudoRNG) SetSeed(seed []byte) error {
	return nil
}

func Test_DbgGraph(t *testing.T) {
	RunDbgGraph(3, NewPseudoRNG, t)
}

/**/
// Test3NodeE2E performs a basic test with 3 simulated nodes. To make
// this work, wrappers around the adapters are introduced to copy
// what would be sent over the network between each stream instead.
func Test3NodeE2E(t *testing.T) {
	//nodeCount := 3
	batchSize := uint32(1)
	grp := cyclic.NewGroup(large.NewIntFromString(TinyStrongPrime, 16),
		large.NewInt(4))
	rngConstructor := NewPseudoRNG // FIXME: Why?
	rngStreamGen := fastRNG.NewStreamGenerator(10000,
		uint(runtime.NumCPU()), csprng.NewSystemRNG)
	registry := createDummyUserList(grp, rngConstructor())
	dummyUser, _ := registry.GetUser(id.NewIdFromUInt(uint64(123), id.User, t), grp)

	//make the round buffer and manually set the round keys
	roundBuf := round.NewBuffer(grp, batchSize, batchSize)
	roundBuf.InitLastNode()
	grp.SetBytes(roundBuf.Z, []byte{13})
	tmp := grp.NewInt(1)
	// >>> ((4**13)**13)**13%107
	// 10
	grp.ExpG(roundBuf.Z, roundBuf.CypherPublicKey)     // First node
	grp.Exp(roundBuf.CypherPublicKey, roundBuf.Z, tmp) // Mid
	grp.Exp(tmp, roundBuf.Z, roundBuf.CypherPublicKey) // Last
	grp.SetBytes(roundBuf.R.Get(0), []byte{26})
	grp.SetBytes(roundBuf.Y_R.Get(0), []byte{69})
	grp.SetBytes(roundBuf.S.Get(0), []byte{77})
	grp.SetBytes(roundBuf.Y_S.Get(0), []byte{81})
	grp.SetBytes(roundBuf.U.Get(0), []byte{94})
	grp.SetBytes(roundBuf.Y_U.Get(0), []byte{87})
	grp.SetBytes(roundBuf.V.Get(0), []byte{18})
	grp.SetBytes(roundBuf.Y_V.Get(0), []byte{79})

	t.Logf("Public Key: %v", roundBuf.CypherPublicKey.Bytes())

	streams := make(map[string]*DebugStream)

	dGrph := buildAndStartGraph3(batchSize, grp, roundBuf, registry,
		rngStreamGen, streams, t)
	megaStream := dGrph.GetStream().(*DebugStream)
	streams["END"] = megaStream

	// Create messsages
	megaStream.KeygenDecryptStream.Salts[0] = []byte{0}
	megaStream.KeygenDecryptStream.Users[0] = dummyUser.ID
	ecrPayloadA := grp.NewInt(31)
	ecrPayloadB := grp.NewInt(1)

	// Send message through the graph
	go func() {
		grp.Set(megaStream.DecryptStream.KeysPayloadA.Get(0), grp.NewInt(1))
		grp.Set(megaStream.DecryptStream.KeysPayloadB.Get(0), grp.NewInt(1))
		grp.Set(megaStream.DecryptStream.CypherPayloadA.Get(0),
			grp.NewInt(1))
		grp.Set(megaStream.DecryptStream.CypherPayloadB.Get(0), grp.NewInt(1))

		grp.Set(megaStream.KeygenDecryptStream.EcrPayloadA.Get(0), ecrPayloadA)
		grp.Set(megaStream.KeygenDecryptStream.EcrPayloadB.Get(0), ecrPayloadB)
		grp.Set(megaStream.KeygenDecryptStream.KeysPayloadA.Get(0),
			grp.NewInt(1))
		grp.Set(megaStream.KeygenDecryptStream.KeysPayloadB.Get(0),
			grp.NewInt(1))

		chunk := services.NewChunk(0, 1)
		dGrph.Send(chunk, nil)
	}()

	numDoneSlots := 0
	for chnk, ok := dGrph.GetOutput(); ok; chnk, ok =
		dGrph.GetOutput() {
		for i := chnk.Begin(); i < chnk.End(); i++ {
			numDoneSlots++
			t.Logf("done slot: %d, total done: %d",
				i, numDoneSlots)
		}
	}

	// Compute result directly

	// We're using the same keys for all nodes (this is a shortcut but is OK
	// AND it tests for if round buffer data is overwritten)
	bufs := make([]*round.Buffer, 3)
	bufs[0] = roundBuf
	bufs[1] = roundBuf
	bufs[2] = roundBuf

	PayloadA, PayloadB := ComputePrecomputation(grp, bufs, t)
	ss := streams["END"].StripStream
	if ss.PayloadAPrecomputation.Get(0).Cmp(PayloadA) != 0 {
		t.Errorf("%v != %v",
			ss.PayloadAPrecomputation.Get(0).Bytes(), PayloadA.Bytes())
	}
	if ss.PayloadBPrecomputation.Get(0).Cmp(PayloadB) != 0 {
		t.Errorf("%v != %v",
			ss.PayloadBPrecomputation.Get(0).Bytes(), PayloadB.Bytes())
	}

	// Verify Realtime
	expPayloadA := grp.NewInt(31)
	expPayloadB := grp.NewInt(1)
	is := streams["END"].IdentifyStream
	if is.EcrPayloadAPermuted[0].Cmp(expPayloadA) != 0 {
		t.Errorf("%v != %v", expPayloadA.Bytes(),
			is.EcrPayloadAPermuted[0].Bytes())
	}
	if is.EcrPayloadBPermuted[0].Cmp(expPayloadB) != 0 {
		t.Errorf("%v != %v", expPayloadB.Bytes(),
			is.EcrPayloadBPermuted[0].Bytes())
	}
}

/**/
