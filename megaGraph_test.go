package main

import (
	"fmt"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cmix"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/graphs"
	"gitlab.com/elixxir/server/graphs/precomputation"
	"gitlab.com/elixxir/server/graphs/realtime"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/services"
	"golang.org/x/crypto/blake2b"
	"math/rand"
	"runtime"
	"testing"
)

type MegaStream struct {
	precomputation.GenerateStream
	precomputation.DecryptStream
	precomputation.PermuteStream
	precomputation.StripStream //Strip contains reveal
	realtime.KeygenDecryptStream
	realtime.IdentifyStream //Identify contains permute
}

func (mega *MegaStream) GetName() string {
	return "MegaStream"
}

func (mega *MegaStream) Link(grp *cyclic.Group, batchSize uint32, source ...interface{}) {
	roundBuf := source[0].(*round.Buffer)
	userRegistry := source[1].(*server.Instance).GetUserRegistry()
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
	mega.LinkGenerateStream(grp, batchSize, roundBuf, rngConstructor)
	mega.LinkPrecompDecryptStream(grp, batchSize, roundBuf, keysMsg, cypherMsg, keysAD, cypherAD)
	mega.LinkPrecompPermuteStream(grp, batchSize, roundBuf, keysMsg, cypherMsg, keysAD, cypherAD,
		keysMsgPermuted, cypherMsgPermuted, keysADPermuted, cypherADPermuted)
	mega.LinkPrecompStripStream(grp, batchSize, roundBuf, cypherMsg, cypherAD)

	//Generate Passthroughs for realtime
	ecrMsg := grp.NewIntBuffer(batchSize, grp.NewInt(1))
	ecrAD := grp.NewIntBuffer(batchSize, grp.NewInt(1))
	ecrMsgPermuted := make([]*cyclic.Int, batchSize)
	ecrADPermuted := make([]*cyclic.Int, batchSize)
	users := make([]*id.User, batchSize)

	for i := uint32(0); i < batchSize; i++ {
		users[i] = &id.User{}
	}

	mega.LinkRealtimeDecryptStream(grp, batchSize, roundBuf,
		userRegistry, ecrMsg, ecrAD, grp.NewIntBuffer(batchSize, grp.NewInt(1)),
		grp.NewIntBuffer(batchSize, grp.NewInt(1)), users, make([][]byte, batchSize))

	mega.LinkIdentifyStreams(grp, batchSize, roundBuf, ecrMsg, ecrAD, ecrMsgPermuted, ecrADPermuted)
}

func (*MegaStream) Input(index uint32, slot *mixmessages.Slot) error {
	return nil
}

func (*MegaStream) Output(index uint32) *mixmessages.Slot {
	return nil
}

var ReintegratePrecompPermute = services.Module{
	Adapt: func(stream services.Stream, cryptop cryptops.Cryptop, chunk services.Chunk) error {
		mega, ok := stream.(*MegaStream)

		if !ok {
			return services.InvalidTypeAssert
		}

		ppsi := mega.PermuteStream

		for i := chunk.Begin(); i < chunk.End(); i++ {
			ppsi.Grp.Set(ppsi.KeysMsg.Get(i), ppsi.KeysMsgPermuted[i])
			ppsi.Grp.Set(ppsi.CypherMsg.Get(i), ppsi.CypherMsgPermuted[i])
			ppsi.Grp.Set(ppsi.KeysAD.Get(i), ppsi.KeysADPermuted[i])
			ppsi.Grp.Set(ppsi.CypherAD.Get(i), ppsi.CypherADPermuted[i])

		}
		return nil
	},
	Cryptop:        cryptops.Mul2,
	NumThreads:     services.AutoNumThreads,
	InputSize:      services.AutoInputSize,
	Name:           "PrecompPermuteReintegration",
	StartThreshold: 1.0,
}

func InitMegaGraph(gc services.GraphGenerator) *services.Graph {
	g := gc.NewGraph("MegaGraph", &MegaStream{})

	//modules for precomputation
	generate := precomputation.Generate.DeepCopy()
	decryptElgamal := precomputation.DecryptElgamal.DeepCopy()
	permuteElgamal := precomputation.PermuteElgamal.DeepCopy()
	permuteReintegrate := ReintegratePrecompPermute.DeepCopy()
	revealRoot := precomputation.RevealRootCoprime.DeepCopy()
	stripInverse := precomputation.StripInverse.DeepCopy()
	stripMul2 := precomputation.StripMul2.DeepCopy()

	//modules for real time
	decryptKeygen := graphs.Keygen.DeepCopy()
	decryptMul3 := realtime.DecryptMul3.DeepCopy()
	permuteMul2 := realtime.PermuteMul2.DeepCopy()
	identifyMal2 := realtime.IdentifyMul2.DeepCopy()

	g.First(generate)
	g.Connect(generate, decryptElgamal)
	g.Connect(decryptElgamal, permuteElgamal)
	g.Connect(permuteElgamal, permuteReintegrate)
	g.Connect(permuteReintegrate, revealRoot)
	g.Connect(revealRoot, stripInverse)
	g.Connect(stripInverse, stripMul2)
	g.Connect(stripMul2, decryptKeygen)
	g.Connect(decryptKeygen, decryptMul3)
	g.Connect(decryptMul3, permuteMul2)
	g.Connect(permuteMul2, identifyMal2)
	g.Last(identifyMal2)

	return g
}

func RunMegaGraph(batchSize uint32, rngConstructor func() csprng.Source, t *testing.T) {

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

	instance := server.CreateServerInstance(grp, &globals.UserMap{})

	registry := instance.GetUserRegistry()
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
			t.Error("MegaGraph: could not rng")
		}
		baseKeyBytes[len(baseKeyBytes)-1] |= 0x01
		u.BaseKey = grp.NewIntFromBytes(baseKeyBytes)
		registry.UpsertUser(u)

		salt := make([]byte, 32)
		_, err = rng.Read(salt)
		if err != nil {
			t.Error("MegaGraph: could not rng")
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
			t.Error("MegaGraph: could not rng")
		}
		messageBytes[len(messageBytes)-1] |= 0x01
		messageList = append(messageList, grp.NewIntFromBytes(messageBytes))

		adBytes := make([]byte, 32)
		_, err = rng.Read(adBytes)
		if err != nil {
			t.Error("MegaGraph: could not rng")
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
		keyMsg := cmix.ClientKeyGen(grp, salts[i], []*cyclic.Int{userList[i].BaseKey})

		hash.Reset()
		hash.Write(salts[i])

		ADMsg := cmix.ClientKeyGen(grp, hash.Sum(nil), []*cyclic.Int{userList[i].BaseKey})

		ecrMsgs = append(ecrMsgs, grp.Mul(messageList[i], keyMsg, grp.NewInt(1)))
		ecrAD = append(ecrAD, grp.Mul(ADList[i], ADMsg, grp.NewInt(1)))
	}

	//make the graph
	PanicHandler := func(g, m string, err error) {
		panic(fmt.Sprintf("Error in module %s of graph %s: %s", g, m, err.Error()))
	}

	gc := services.NewGraphGenerator(4, PanicHandler, uint8(runtime.NumCPU()), services.AutoOutputSize, 0)
	megaGraph := InitMegaGraph(gc)

	megaGraph.Build(batchSize)

	//make the round buffer
	roundBuf := round.NewBuffer(grp, batchSize, megaGraph.GetExpandedBatchSize())
	roundBuf.InitLastNode()

	//do a mock share phase
	zBytes := make([]byte, 31)
	rng.Read(zBytes)
	zBytes[0] |= 0x01
	zBytes[len(zBytes)-1] |= 0x01

	grp.SetBytes(roundBuf.Z, zBytes)
	grp.ExpG(roundBuf.Z, roundBuf.CypherPublicKey)

	megaGraph.Link(grp, roundBuf, instance, rngConstructor)

	stream := megaGraph.GetStream()

	megaStream := stream.(*MegaStream)

	megaGraph.Run()

	go func() {
		t.Log("Beginning test")
		for i := uint32(0); i < batchSize; i++ {
			megaStream.KeygenDecryptStream.Salts[i] = salts[i]
			megaStream.KeygenDecryptStream.Users[i].SetBytes((userList[i].ID)[:])
			grp.Set(megaStream.IdentifyStream.EcrMsg.Get(i), ecrMsgs[i])
			grp.Set(megaStream.IdentifyStream.EcrAD.Get(i), ecrAD[i])
			chunk := services.NewChunk(i, i+1)
			megaGraph.Send(chunk)
		}
	}()

	numDoneSlots := 0

	for chunk, ok := megaGraph.GetOutput(); ok; chunk, ok = megaGraph.GetOutput() {
		for i := chunk.Begin(); i < chunk.End(); i++ {
			numDoneSlots++
			fmt.Println("done slot:", i, " total done:", numDoneSlots)
		}
	}

	for i := uint32(0); i < batchSize; i++ {
		if megaStream.IdentifyStream.EcrMsg.Get(i).Cmp(messageList[i]) != 0 {
			t.Errorf("MegaGraph: Decrypted message not the same as send message on slot %v, "+
				"Sent: %s, Decrypted: %s", i, messageList[i].Text(16),
				megaStream.IdentifyStream.EcrMsg.Get(i).Text(16))
		}
		if megaStream.IdentifyStream.EcrAD.Get(i).Cmp(ADList[i]) != 0 {
			t.Errorf("MegaGraph: Decrypted AD not the same as send message on slot %v, "+
				"Sent: %s, Decrypted: %s", i, ADList[i].Text(16),
				megaStream.IdentifyStream.EcrAD.Get(i).Text(16))
		}
	}

}

func Test_MegaStream(t *testing.T) {
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

	batchSize := uint32(1000)

	PanicHandler := func(g, m string, err error) {
		panic(fmt.Sprintf("Error in module %s of graph %s: %s", g, m, err.Error()))
	}

	gc := services.NewGraphGenerator(4, PanicHandler, uint8(runtime.NumCPU()), services.AutoOutputSize, 0)
	megaGraph := InitMegaGraph(gc)

	megaGraph.Build(batchSize)

	//make the round buffer
	roundBuf := round.NewBuffer(grp, batchSize, megaGraph.GetExpandedBatchSize())

	instance := server.CreateServerInstance(grp, &globals.UserMap{})

	megaGraph.Link(grp, roundBuf, instance, csprng.NewSystemRNG)

	stream := megaGraph.GetStream()

	_, ok := stream.(precomputation.GenerateSubstreamInterface)

	if !ok {
		t.Errorf("MegaStream: type assert failed when getting 'GenerateSubstreamInterface'")
	}

	_, ok = stream.(precomputation.PrecompDecryptSubstreamInterface)

	if !ok {
		t.Errorf("MegaStream: type assert failed when getting 'PrecompDecryptSubstreamInterface'")
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

func Test_MegaGraph(t *testing.T) {
	RunMegaGraph(1000, NewPsudoRNG, t)
}
