package main

import (
	"fmt"
	"github.com/jinzhu/copier"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cmix"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/graphs"
	"gitlab.com/elixxir/server/graphs/precomputation"
	"gitlab.com/elixxir/server/graphs/realtime"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/services"
	"golang.org/x/crypto/blake2b"
	"math/rand"
	"runtime"
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
// single node system. In other words, it multiplies the R, S, T
// keys together for the message precomputation, and it does the same for
// the U, V keys to make the associated data precomputation.
func ComputeSingleNodePrecomputation(grp *cyclic.Group, round *round.Buffer) (
	*cyclic.Int, *cyclic.Int) {
	MP := grp.NewInt(1)

	keys := round

	rInv := grp.NewMaxInt()
	sInv := grp.NewMaxInt()
	uInv := grp.NewMaxInt()
	vInv := grp.NewMaxInt()

	grp.Inverse(keys.R.Get(0), rInv)
	grp.Inverse(keys.S.Get(0), sInv)
	grp.Inverse(keys.U.Get(0), uInv)
	grp.Inverse(keys.V.Get(0), vInv)

	grp.Mul(MP, rInv, MP)
	grp.Mul(MP, sInv, MP)

	RP := grp.NewInt(1)

	grp.Mul(RP, uInv, RP)
	grp.Mul(RP, vInv, RP)

	return MP, RP

}

// Compute Precomputation for N nodes
// NOTE: This does not handle precomputation under permutation, but it will
//       handle multi-node precomputation checks.
func ComputePrecomputation(grp *cyclic.Group, rounds []*round.Round) (
	*cyclic.Int, *cyclic.Int) {
	MP := grp.NewInt(1)
	RP := grp.NewInt(1)
	rInv := grp.NewMaxInt()
	sInv := grp.NewMaxInt()
	uInv := grp.NewMaxInt()
	vInv := grp.NewMaxInt()

	for i := range rounds {
		keys := rounds[i].GetBuffer()
		grp.Inverse(keys.R.Get(0), rInv)
		grp.Inverse(keys.S.Get(0), sInv)
		grp.Inverse(keys.U.Get(0), uInv)
		grp.Inverse(keys.V.Get(0), vInv)

		grp.Mul(MP, rInv, MP)
		grp.Mul(MP, sInv, MP)

		grp.Mul(RP, uInv, RP)
		grp.Mul(RP, vInv, RP)
	}
	return MP, RP
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
	u.ID = id.NewUserFromUint(uint64(123), nil)
	baseKeyBytes := []byte{1}
	u.BaseKey = grp.NewIntFromBytes(baseKeyBytes)
	// FIXME: This should really not be necessary and this API is wonky
	dsaParams := signature.GetDefaultDSAParams()
	dsaPrivateKey := dsaParams.PrivateKeyGen(rng)
	u.PublicKey = dsaPrivateKey.PublicKeyGen()
	registry.UpsertUser(u)
	userList = append(userList, u)
	return registry
}

func buildAndStartGraph(batchSize uint32, grp *cyclic.Group,
	roundBuf *round.Buffer, registry *globals.UserMap,
	rngConstructor func() csprng.Source, streams map[string]*DebugStream,
	t *testing.T) *services.Graph {
	//make the graph
	PanicHandler := func(g, m string, err error) {
		panic(fmt.Sprintf("Error in module %s of graph %s: %s",
			g, m, err.Error()))
	}
	// NOTE: input size greater than 1 would necessarily cause a hang here
	// since we never send more than 1 message through.
	gc := services.NewGraphGenerator(1, PanicHandler,
		1, 1, 0)
	dGrph := InitDbgGraph(gc, streams, t)
	dGrph.Build(batchSize)

	dGrph.Link(grp, roundBuf, registry, rngConstructor)
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
// meaning the Associated Data representation in the group
// would be much bigger than the hardcoded P value of 107
// Secondly, the first byte of the Associated Data is randomly generated,
// so the expected values throughout the pipeline would need to be calculated
// Not having proper Associated Data is not an issue in this particular test,
// because here only cryptops are chained
// The actual extraction of recipientID from associated data only occurs in
// handlers from the io package
func TestEndToEndCryptops(t *testing.T) {
	// Init, we use a small prime to make it easier to run the numbers
	// when debugging
	grp := cyclic.NewGroup(large.NewIntFromString(TinyStrongPrime, 16),
		large.NewInt(4), large.NewInt(5))

	rngConstructor := NewPsudoRNG // FIXME: Why?
	batchSize := uint32(1)

	registry := createDummyUserList(grp, rngConstructor())
	dummyUser, _ := registry.GetUser(id.NewUserFromUint(uint64(123), t))

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
		rngConstructor, streams, t)
	megaStream := dGrph.GetStream().(*DebugStream)
	streams["END"] = megaStream

	// Create messsages
	megaStream.KeygenDecryptStream.Salts[0] = []byte{0}
	megaStream.KeygenDecryptStream.Users[0] = dummyUser.ID
	ecrMsg := grp.NewInt(31)
	ecrAD := grp.NewInt(1)

	// Send message through the graph
	go func() {
		grp.Set(megaStream.DecryptStream.KeysMsg.Get(0), grp.NewInt(1))
		grp.Set(megaStream.DecryptStream.KeysAD.Get(0), grp.NewInt(1))
		grp.Set(megaStream.DecryptStream.CypherMsg.Get(0),
			grp.NewInt(1))
		grp.Set(megaStream.DecryptStream.CypherAD.Get(0), grp.NewInt(1))

		grp.Set(megaStream.KeygenDecryptStream.EcrMsg.Get(0), ecrMsg)
		grp.Set(megaStream.KeygenDecryptStream.EcrAD.Get(0), ecrAD)
		grp.Set(megaStream.KeygenDecryptStream.KeysMsg.Get(0),
			grp.NewInt(1))
		grp.Set(megaStream.KeygenDecryptStream.KeysAD.Get(0),
			grp.NewInt(1))

		chunk := services.NewChunk(0, 1)
		dGrph.Send(chunk)
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
	            KeysMsg/AD: ([5] / [17]), CypherMsg/AD: ([79] / [36])
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
	    dGrph_test.go:428: MP: 69 in GRP: xjz30UG9n4...,
		RP: 16 in GRP: xjz30UG9n4...
	PASS
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
	if ds.KeysMsg.Get(0).Cmp(grp.NewInt(5)) != 0 {
		t.Errorf("Precomp Decrypt KeysMsg: %v != [5]",
			ds.KeysMsg.Get(0).Bytes())
	}
	if ds.KeysAD.Get(0).Cmp(grp.NewInt(17)) != 0 {
		t.Errorf("Precomp Decrypt KeysAD: %v != [17]",
			ds.KeysAD.Get(0).Bytes())
	}
	if ds.CypherMsg.Get(0).Cmp(grp.NewInt(79)) != 0 {
		t.Errorf("Precomp Decrypt CypherMsg: %v != [79]",
			ds.CypherMsg.Get(0).Bytes())
	}
	if ds.CypherAD.Get(0).Cmp(grp.NewInt(36)) != 0 {
		t.Errorf("Precomp Decrypt CypherAD: %v != [36]",
			ds.CypherAD.Get(0).Bytes())
	}
	*/

	// Compute result directly
	MP, RP := ComputeSingleNodePrecomputation(grp, roundBuf)
	t.Logf("MP: %s, RP: %s",
		MP.Text(10), RP.Text(10))
	ss := streams["END"].StripStream
	if ss.MessagePrecomputation.Get(0).Cmp(MP) != 0 {
		t.Errorf("%v != %v",
			ss.MessagePrecomputation.Get(0).Bytes(), MP.Bytes())
	}
	if ss.ADPrecomputation.Get(0).Cmp(RP) != 0 {
		t.Errorf("%v != %v",
			ss.ADPrecomputation.Get(0).Bytes(), RP.Bytes())
	}

	/* Most of these are incorrect because we changed the computation to
	   2 keys instead of 3 as well as flipped to using inverse on clients
	expectedRTDecrypt := []*cyclic.Int{
		// 57 for Msg and 94 for AD
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

	expMsg := grp.NewInt(31)
	expAD := grp.NewInt(1)
	is := streams["END"].IdentifyStream
	if is.EcrMsgPermuted[0].Cmp(expMsg) != 0 {
		t.Errorf("%v != %v", expMsg.Bytes(),
			megaStream.IdentifyStream.EcrMsgPermuted[0].Bytes())
	}
	if is.EcrADPermuted[0].Cmp(expAD) != 0 {
		t.Errorf("%v != %v", expAD.Bytes(),
			megaStream.IdentifyStream.EcrADPermuted[0].Bytes())
	}

}

// TestBatchSize3 runs the End to End test with 3 messages instead of 1
func TestBatchSize3(t *testing.T) {
	// Init, we use a small prime to make it easier to run the numbers
	// when debugging
	grp := cyclic.NewGroup(large.NewIntFromString(TinyStrongPrime, 16),
		large.NewInt(4), large.NewInt(5))

	rngConstructor := NewPsudoRNG // FIXME: Why?
	batchSize := uint32(4)

	registry := createDummyUserList(grp, rngConstructor())
	dummyUser, _ := registry.GetUser(id.NewUserFromUint(uint64(123), t))

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
	streams := make(map[string]*DebugStream)

	dGrph := buildAndStartGraph(batchSize, grp, roundBuf, registry,
		rngConstructor, streams, t)
	megaStream := dGrph.GetStream().(*DebugStream)
	streams["END"] = megaStream

	// Create messsages
	for i := uint32(0); i < batchSize; i++ {
		megaStream.KeygenDecryptStream.Salts[i] = []byte{0}
		megaStream.KeygenDecryptStream.Users[i] = dummyUser.ID
	}

	// Send message through the graph
	go func() {
		for i := uint32(0); i < batchSize; i++ {
			ecrMsg := grp.NewInt((30+int64(i))%106 + 1)
			ecrAD := grp.NewInt((int64(i))%106 + 1)
			grp.Set(megaStream.DecryptStream.KeysMsg.Get(i),
				grp.NewInt(1))
			grp.Set(megaStream.DecryptStream.KeysAD.Get(i),
				grp.NewInt(1))
			grp.Set(megaStream.DecryptStream.CypherMsg.Get(i),
				grp.NewInt(1))
			grp.Set(megaStream.DecryptStream.CypherAD.Get(i),
				grp.NewInt(1))

			grp.Set(megaStream.KeygenDecryptStream.EcrMsg.Get(i),
				ecrMsg)
			grp.Set(megaStream.KeygenDecryptStream.EcrAD.Get(i),
				ecrAD)
			grp.Set(megaStream.KeygenDecryptStream.KeysMsg.Get(i),
				grp.NewInt(1))
			grp.Set(megaStream.KeygenDecryptStream.KeysAD.Get(i),
				grp.NewInt(1))

			chunk := services.NewChunk(i, i+1)
			dGrph.Send(chunk)
		}
	}()

	numDoneSlots := 0
	for chunk, ok := dGrph.GetOutput(); ok; chunk, ok =
		dGrph.GetOutput() {
		for i := chunk.Begin(); i < chunk.End(); i++ {
			numDoneSlots++
		}
	}

	// Compute result directly
	MP, RP := ComputeSingleNodePrecomputation(grp, roundBuf)
	ss := streams["END"].StripStream
	is := streams["END"].IdentifyStream
	for i := uint32(0); i < batchSize; i++ {
		// Verify precomputation
		if ss.MessagePrecomputation.Get(i).Cmp(MP) != 0 {
			t.Errorf("%v != %v",
				ss.MessagePrecomputation.Get(i).Bytes(),
				MP.Bytes())
		}
		if ss.ADPrecomputation.Get(i).Cmp(RP) != 0 {
			t.Errorf("%v != %v",
				ss.ADPrecomputation.Get(i).Bytes(), RP.Bytes())
		}

		// Verify Realtime
		for i := uint32(0); i < batchSize; i++ {
			expMsg := grp.NewInt(int64(30 + (3+i)%batchSize + 1))
			expAD := grp.NewInt(int64((3+i)%batchSize + 1))
			if is.EcrMsgPermuted[i].Cmp(expMsg) != 0 {
				t.Errorf("%v != %v", expMsg.Bytes(),
					is.EcrMsgPermuted[i].Bytes())
			}
			if is.EcrADPermuted[i].Cmp(expAD) != 0 {
				t.Errorf("%v != %v", expAD.Bytes(),
					is.EcrADPermuted[i].Bytes())
			}
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
	rngConstructor := source[2].(func() csprng.Source)

	//Generate passthroughs for precomputation
	keysMsg := grp.NewIntBuffer(batchSize, grp.NewInt(1))
	cypherMsg := grp.NewIntBuffer(batchSize, grp.NewInt(1))
	keysAD := grp.NewIntBuffer(batchSize, grp.NewInt(1))
	cypherAD := grp.NewIntBuffer(batchSize, grp.NewInt(1))

	keysMsgPermuted := make([]*cyclic.Int, batchSize)
	cypherMsgPermuted := make([]*cyclic.Int, batchSize)
	keysADPermuted := make([]*cyclic.Int, batchSize)
	cypherADPermuted := make([]*cyclic.Int, batchSize)

	//Link precomputation
	ds.LinkGenerateStream(grp, batchSize, roundBuf, rngConstructor)
	ds.LinkPrecompDecryptStream(grp, batchSize, roundBuf, keysMsg,
		cypherMsg, keysAD, cypherAD)
	ds.LinkPrecompPermuteStream(grp, batchSize, roundBuf, keysMsg,
		cypherMsg, keysAD, cypherAD, keysMsgPermuted, cypherMsgPermuted,
		keysADPermuted, cypherADPermuted)
	ds.LinkPrecompStripStream(grp, batchSize, roundBuf, cypherMsg,
		cypherAD, keysMsg, keysAD)

	//Generate Passthroughs for realtime
	ecrMsg := grp.NewIntBuffer(batchSize, grp.NewInt(1))
	ecrAD := grp.NewIntBuffer(batchSize, grp.NewInt(1))
	ecrMsgPermuted := make([]*cyclic.Int, batchSize)
	ecrADPermuted := make([]*cyclic.Int, batchSize)
	users := make([]*id.User, batchSize)

	for i := uint32(0); i < batchSize; i++ {
		users[i] = &id.User{}
	}

	ds.LinkRealtimeDecryptStream(grp, batchSize, roundBuf,
		userRegistry, ecrMsg, ecrAD, grp.NewIntBuffer(batchSize,
			grp.NewInt(1)),
		grp.NewIntBuffer(batchSize, grp.NewInt(1)), users,
		make([][]byte, batchSize))

	ds.LinkIdentifyStreams(grp, batchSize, roundBuf, ecrMsg, ecrAD,
		ecrMsgPermuted, ecrADPermuted)
}

func (ds *DebugStream) Input(index uint32, slot *mixmessages.Slot) error {
	es := make([]error, 6)
	es[0] = ds.GenerateStream.Input(index, slot)
	es[1] = ds.DecryptStream.Input(index, slot)
	es[2] = ds.PermuteStream.Input(index, slot)
	es[3] = ds.StripStream.Input(index, slot)
	es[4] = ds.KeygenDecryptStream.Input(index, slot)
	es[5] = ds.IdentifyStream.Input(index, slot)

	var lastErr error
	for i := 0; i < len(es); i++ {
		if es[i] != nil {
			fmt.Printf("Error DebugStream Input: %v\n", es[i])
			lastErr = es[i]
		}
	}
	return lastErr
}

func (ds *DebugStream) Output(index uint32) *mixmessages.Slot {
	ds.Outputs = make([]*mixmessages.Slot, 6)
	ds.Outputs[0] = ds.GenerateStream.Output(index)
	ds.Outputs[1] = ds.DecryptStream.Output(index)
	ds.Outputs[2] = ds.PermuteStream.Output(index)
	ds.Outputs[3] = ds.StripStream.Output(index)
	ds.Outputs[4] = ds.KeygenDecryptStream.Output(index)
	ds.Outputs[5] = ds.IdentifyStream.Output(index)
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
			ppsi.Grp.Set(ppsi.KeysMsg.Get(i),
				ppsi.KeysMsgPermuted[i])
			ppsi.Grp.Set(ppsi.CypherMsg.Get(i),
				ppsi.CypherMsgPermuted[i])
			ppsi.Grp.Set(ppsi.KeysAD.Get(i),
				ppsi.KeysADPermuted[i])
			ppsi.Grp.Set(ppsi.CypherAD.Get(i),
				ppsi.CypherADPermuted[i])
		}
		return nil
	},
	Cryptop:        cryptops.Mul2,
	NumThreads:     services.AutoNumThreads,
	InputSize:      services.AutoInputSize,
	Name:           "PrecompPermuteReintegration",
	StartThreshold: 1.0,
}

func InitDbgGraph(gc services.GraphGenerator, streams map[string]*DebugStream,
	t *testing.T) *services.Graph {
	g := gc.NewGraph("DbgGraph", &DebugStream{})

	//modules for precomputation
	//generate := precomputation.Generate.DeepCopy()
	decryptElgamal := precomputation.DecryptElgamal.DeepCopy()
	permuteElgamal := precomputation.PermuteElgamal.DeepCopy()
	permuteReintegrate := ReintegratePrecompPermute.DeepCopy()
	revealRoot := precomputation.RevealRootCoprime.DeepCopy()
	stripInverse := precomputation.StripInverse.DeepCopy()
	stripMul2 := precomputation.StripMul2.DeepCopy()

	//modules for real time
	//decryptKeygen := DummyKeygen.DeepCopy()
	decryptMul3 := realtime.DecryptMul3.DeepCopy()
	permuteMul2 := realtime.PermuteMul2.DeepCopy()
	identifyMal2 := realtime.IdentifyMul2.DeepCopy()

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
	g.Connect(dPPermuteMul2, identifyMal2)
	g.Last(identifyMal2)
	return g
}

func RunDbgGraph(batchSize uint32, rngConstructor func() csprng.Source,
	t *testing.T) {
	grp := cyclic.NewGroup(large.NewIntFromString(MODP768, 16),
		large.NewInt(2), large.NewInt(1283))

	//nid := server.GenerateId()

	//instance := server.CreateServerInstance(grp, nid, &globals.UserMap{})

	registry := &globals.UserMap{}
	var userList []*globals.User

	var salts [][]byte

	rng := rngConstructor()

	//make the user IDs and their base keys and the salts
	for i := uint32(0); i < batchSize; i++ {
		u := registry.NewUser(grp)
		u.ID = id.NewUserFromUint(uint64(i), t)

		baseKeyBytes := make([]byte, 32)
		_, err := rng.Read(baseKeyBytes)
		if err != nil {
			t.Error("DbgGraph: could not rng")
		}
		baseKeyBytes[len(baseKeyBytes)-1] |= 0x01
		u.BaseKey = grp.NewIntFromBytes(baseKeyBytes)
		registry.UpsertUser(u)

		salt := make([]byte, 32)
		_, err = rng.Read(salt)
		if err != nil {
			t.Error("DbgGraph: could not rng")
		}
		salts = append(salts, salt)

		userList = append(userList, u)
	}

	var messageList []*cyclic.Int
	var ADList []*cyclic.Int

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
		ADList = append(ADList, grp.NewIntFromBytes(adBytes))
	}

	hash, err := blake2b.New256(nil)

	if err != nil {
		t.Errorf("Could not get blake2b hash: %s", err.Error())
	}

	var ecrMsgs []*cyclic.Int
	var ecrAD []*cyclic.Int

	//encrypt the messages
	for i := uint32(0); i < batchSize; i++ {
		keyMsg := cmix.ClientKeyGen(grp, salts[i],
			[]*cyclic.Int{userList[i].BaseKey})

		hash.Reset()
		hash.Write(salts[i])

		ADMsg := cmix.ClientKeyGen(grp, hash.Sum(nil),
			[]*cyclic.Int{userList[i].BaseKey})

		ecrMsgs = append(ecrMsgs, grp.Mul(messageList[i], keyMsg,
			grp.NewInt(1)))
		ecrAD = append(ecrAD, grp.Mul(ADList[i], ADMsg, grp.NewInt(1)))
	}

	//make the graph
	PanicHandler := func(g, m string, err error) {
		panic(fmt.Sprintf("Error in module %s of graph %s: %s",
			g, m, err.Error()))
	}

	gc := services.NewGraphGenerator(4, PanicHandler,
		uint8(runtime.NumCPU()), services.AutoOutputSize, 0)
	streams := make(map[string]*DebugStream)
	dGrph := InitDbgGraph(gc, streams, t)

	dGrph.Build(batchSize)

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

	dGrph.Link(grp, roundBuf, registry, rngConstructor)

	stream := dGrph.GetStream()

	megaStream := stream.(*DebugStream)

	dGrph.Run()

	go func() {
		t.Log("Beginning test")
		for i := uint32(0); i < batchSize; i++ {
			megaStream.KeygenDecryptStream.Salts[i] = salts[i]
			megaStream.KeygenDecryptStream.Users[i] = userList[i].ID
			grp.Set(megaStream.IdentifyStream.EcrMsg.Get(i),
				ecrMsgs[i])
			grp.Set(megaStream.IdentifyStream.EcrAD.Get(i),
				ecrAD[i])
			chunk := services.NewChunk(i, i+1)
			dGrph.Send(chunk)
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
		if megaStream.IdentifyStream.EcrMsg.Get(i).Cmp(
			messageList[i]) != 0 {
			t.Errorf("DbgGraph: Decrypted message not"+
				" the same as message on slot %v,"+
				"Sent: %s, Decrypted: %s", i,
				messageList[i].Text(16),
				megaStream.IdentifyStream.EcrMsg.Get(
					i).Text(16))
		}
		if megaStream.IdentifyStream.EcrAD.Get(i).Cmp(ADList[i]) != 0 {
			t.Errorf("DbgGraph: Decrypted AD not the same"+
				" as send message on slot %v, "+
				"Sent: %s, Decrypted: %s", i,
				ADList[i].Text(16),
				megaStream.IdentifyStream.EcrAD.Get(i).Text(16))
		}
	}

}

func Test_DebugStream(t *testing.T) {
	grp := cyclic.NewGroup(large.NewIntFromString(MODP768, 16),
		large.NewInt(2), large.NewInt(1283))

	batchSize := uint32(1000)

	PanicHandler := func(g, m string, err error) {
		panic(fmt.Sprintf("Error in module %s of graph %s: %s",
			g, m, err.Error()))
	}

	gc := services.NewGraphGenerator(4, PanicHandler,
		uint8(runtime.NumCPU()), services.AutoOutputSize, 0)
	streams := make(map[string]*DebugStream)
	dGrph := InitDbgGraph(gc, streams, t)

	dGrph.Build(batchSize)

	//make the round buffer
	roundBuf := round.NewBuffer(grp, batchSize,
		dGrph.GetExpandedBatchSize())

	dGrph.Link(grp, roundBuf, &globals.UserMap{}, csprng.NewSystemRNG)

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

func NewPsudoRNG() csprng.Source {
	return &PsudoRNG{
		r: rand.New(rand.NewSource(42)),
	}
}

type PsudoRNG struct {
	r *rand.Rand
}

// Read calls the crypto/rand Read function and returns the values
func (p *PsudoRNG) Read(b []byte) (int, error) {
	return p.r.Read(b)
}

// SetSeed has not effect on the system reader
func (p *PsudoRNG) SetSeed(seed []byte) error {
	return nil
}

/*
func Test_DbgGraph(t *testing.T) {
	RunDbgGraph(3, NewPsudoRNG, t)
}
*/

// Test3NodeE2E performs a basic test with 3 simulated nodes. To make
// this work, wrappers around the adapters are introduced to copy
// what would be sent over the network between each stream instead.
func Test3NodeE2E(t *testing.T) {
	//nodeCount := 3
	batchSize := uint32(1)
	grp := cyclic.NewGroup(large.NewIntFromString(TinyStrongPrime, 16),
		large.NewInt(4), large.NewInt(5))
	rngConstructor := NewPsudoRNG // FIXME: Why?
	registry := createDummyUserList(grp, rngConstructor())
	dummyUser, _ := registry.GetUser(id.NewUserFromUint(uint64(123), t))

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
		rngConstructor, streams, t)
	megaStream := dGrph.GetStream().(*DebugStream)
	streams["END"] = megaStream

	// Create messsages
	megaStream.KeygenDecryptStream.Salts[0] = []byte{0}
	megaStream.KeygenDecryptStream.Users[0] = dummyUser.ID
	ecrMsg := grp.NewInt(31)
	ecrAD := grp.NewInt(1)

	// Send message through the graph
	go func() {
		grp.Set(megaStream.DecryptStream.KeysMsg.Get(0), grp.NewInt(1))
		grp.Set(megaStream.DecryptStream.KeysAD.Get(0), grp.NewInt(1))
		grp.Set(megaStream.DecryptStream.CypherMsg.Get(0),
			grp.NewInt(1))
		grp.Set(megaStream.DecryptStream.CypherAD.Get(0), grp.NewInt(1))

		grp.Set(megaStream.KeygenDecryptStream.EcrMsg.Get(0), ecrMsg)
		grp.Set(megaStream.KeygenDecryptStream.EcrAD.Get(0), ecrAD)
		grp.Set(megaStream.KeygenDecryptStream.KeysMsg.Get(0),
			grp.NewInt(1))
		grp.Set(megaStream.KeygenDecryptStream.KeysAD.Get(0),
			grp.NewInt(1))

		chunk := services.NewChunk(0, 1)
		dGrph.Send(chunk)
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
	MP, RP := ComputeSingleNodePrecomputation(grp, roundBuf)
	t.Logf("MP: %s, RP: %s",
		MP.Text(10), RP.Text(10))
	ss := streams["END"].StripStream
	if ss.MessagePrecomputation.Get(0).Cmp(MP) != 0 {
		t.Errorf("%v != %v",
			ss.MessagePrecomputation.Get(0).Bytes(), MP.Bytes())
	}
	if ss.ADPrecomputation.Get(0).Cmp(RP) != 0 {
		t.Errorf("%v != %v",
			ss.ADPrecomputation.Get(0).Bytes(), RP.Bytes())
	}

	// Verify Realtime
	expMsg := grp.NewInt(31)
	expAD := grp.NewInt(1)
	is := streams["END"].IdentifyStream
	if is.EcrMsgPermuted[0].Cmp(expMsg) != 0 {
		t.Errorf("%v != %v", expMsg.Bytes(),
			megaStream.IdentifyStream.EcrMsgPermuted[0].Bytes())
	}
	if is.EcrADPermuted[0].Cmp(expAD) != 0 {
		t.Errorf("%v != %v", expAD.Bytes(),
			megaStream.IdentifyStream.EcrADPermuted[0].Bytes())
	}
}
