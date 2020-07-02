///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package precomputation

import (
	"fmt"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/crypto/shuffle"
	"gitlab.com/elixxir/server/graphs"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/services"
	"reflect"
	"runtime"
	"sync/atomic"
	"testing"
)

// Test that PermuteStream.GetName() returns the correct name
func TestPermuteStream_GetName(t *testing.T) {
	expected := "PrecompPermuteStream"

	stream := PermuteStream{}

	if stream.GetName() != expected {
		t.Errorf("PermuteStream.GetName(), Expected %s, Received %s", expected, stream.GetName())
	}
}

// Test that PermuteStream.Link() Links correctly
func TestPermuteStream_Link(t *testing.T) {
	grp := initPermuteGroup()

	stream := PermuteStream{}

	batchSize := uint32(100)

	roundBuffer := initPermuteRoundBuffer(grp, batchSize)

	stream.Link(grp, batchSize, roundBuffer)

	checkStreamIntBuffer(grp, stream.S, roundBuffer.S, "S", t)
	checkStreamIntBuffer(grp, stream.V, roundBuffer.V, "V", t)
	checkStreamIntBuffer(grp, stream.Y_S, roundBuffer.Y_S, "Y_S", t)
	checkStreamIntBuffer(grp, stream.Y_V, roundBuffer.Y_V, "Y_V", t)

	checkIntBuffer(stream.KeysPayloadA, batchSize, "KeysPayloadA", grp.NewInt(1), t)
	checkIntBuffer(stream.CypherPayloadA, batchSize, "CypherPayloadA", grp.NewInt(1), t)
	checkIntBuffer(stream.KeysPayloadB, batchSize, "KeysPayloadB", grp.NewInt(1), t)
	checkIntBuffer(stream.CypherPayloadB, batchSize, "CypherPayloadB", grp.NewInt(1), t)
}

// Tests Input's happy path
func TestPermuteStream_Input(t *testing.T) {
	grp := initPermuteGroup()

	stream := PermuteStream{}

	batchSize := uint32(100)

	roundBuffer := initPermuteRoundBuffer(grp, batchSize)

	stream.Link(grp, batchSize, roundBuffer)

	for b := uint32(0); b < batchSize; b++ {

		expected := [][]byte{
			{byte(b + 1), 0},
			{byte(b + 1), 1},
			{byte(b + 1), 2},
			{byte(b + 1), 3},
		}

		msg := &mixmessages.Slot{
			EncryptedPayloadAKeys:     expected[0],
			EncryptedPayloadBKeys:     expected[1],
			PartialPayloadACypherText: expected[2],
			PartialPayloadBCypherText: expected[3],
		}

		err := stream.Input(b, msg)
		if err != nil {
			t.Errorf("PermuteStream.Input() errored on slot %v: %s", b, err.Error())
		}

		if !reflect.DeepEqual(stream.KeysPayloadA.Get(b).Bytes(), expected[0]) {
			t.Errorf("PermuteStream.Input() incorrect stored KeysPayloadA data at %v: Expected: %v, Received: %v",
				b, expected[0], stream.KeysPayloadA.Get(b).Bytes())
		}

		if !reflect.DeepEqual(stream.KeysPayloadB.Get(b).Bytes(), expected[1]) {
			t.Errorf("PermuteStream.Input() incorrect stored KeysPayloadB data at %v: Expected: %v, Received: %v",
				b, expected[1], stream.KeysPayloadB.Get(b).Bytes())
		}

		if !reflect.DeepEqual(stream.CypherPayloadA.Get(b).Bytes(), expected[2]) {
			t.Errorf("PermuteStream.Input() incorrect stored CypherPayloadA data at %v: Expected: %v, Received: %v",
				b, expected[2], stream.CypherPayloadA.Get(b).Bytes())
		}

		if !reflect.DeepEqual(stream.CypherPayloadB.Get(b).Bytes(), expected[3]) {
			t.Errorf("PermuteStream.Input() incorrect stored CypherPayloadB data at %v: Expected: %v, Received: %v",
				b, expected[3], stream.CypherPayloadB.Get(b).Bytes())
		}

	}

}

// Tests that the input errors correctly when the index is outside of the batch
func TestPermuteStream_Input_OutOfBatch(t *testing.T) {
	grp := initPermuteGroup()

	stream := PermuteStream{}

	batchSize := uint32(100)

	roundBuffer := initPermuteRoundBuffer(grp, batchSize)

	stream.Link(grp, batchSize, roundBuffer)

	stream.Link(grp, batchSize, roundBuffer)

	msg := &mixmessages.Slot{
		EncryptedPayloadAKeys:     []byte{0},
		EncryptedPayloadBKeys:     []byte{0},
		PartialPayloadACypherText: []byte{0},
		PartialPayloadBCypherText: []byte{0},
	}

	err := stream.Input(batchSize, msg)

	if err == nil {
		t.Errorf("PermuteStream.Input() did nto return an error when out of batch")
	}

	err1 := stream.Input(batchSize+1, msg)

	if err1 == nil {
		t.Errorf("PermuteStream.Input() did nto return an error when out of batch")
	}
}

// Tests that Input errors correct when the passed value is out of the group
func TestPermuteStream_Input_OutOfGroup(t *testing.T) {
	grp := initPermuteGroup()

	stream := PermuteStream{}

	batchSize := uint32(100)

	roundBuffer := initPermuteRoundBuffer(grp, batchSize)

	stream.Link(grp, batchSize, roundBuffer)

	msg := &mixmessages.Slot{
		EncryptedPayloadAKeys:     []byte{0},
		EncryptedPayloadBKeys:     []byte{0},
		PartialPayloadACypherText: []byte{0},
		PartialPayloadBCypherText: []byte{0},
	}

	err := stream.Input(batchSize-10, msg)

	if err != services.ErrOutsideOfGroup {
		t.Errorf("PermuteStream.Input() did not return an error when out of group")
	}
}

// Tests that the output function returns a valid cmixMessage
func TestPermuteStream_Output(t *testing.T) {
	grp := initPermuteGroup()

	stream := PermuteStream{}

	batchSize := uint32(100)

	roundBuffer := initPermuteRoundBuffer(grp, batchSize)

	stream.Link(grp, batchSize, roundBuffer)

	for b := uint32(0); b < batchSize; b++ {

		expected := [][]byte{
			{byte(b + 1), 1},
			{byte(b + 1), 2},
			{byte(b + 1), 3},
			{byte(b + 1), 4},
		}

		stream.KeysPayloadAPermuted[b] = grp.NewIntFromBytes(expected[0])
		stream.KeysPayloadBPermuted[b] = grp.NewIntFromBytes(expected[1])
		stream.CypherPayloadAPermuted[b] = grp.NewIntFromBytes(expected[2])
		stream.CypherPayloadBPermuted[b] = grp.NewIntFromBytes(expected[3])

		output := stream.Output(uint32(b))

		if !reflect.DeepEqual(output.EncryptedPayloadAKeys, expected[0]) {
			t.Errorf("PermuteStream.Output() incorrect received KeysPayloadA data at %v: Expected: %v, Received: %v",
				b, expected[0], output.EncryptedPayloadAKeys)
		}

		if !reflect.DeepEqual(output.EncryptedPayloadBKeys, expected[1]) {
			t.Errorf("PermuteStream.Output() incorrect received KeysPayloadB data at %v: Expected: %v, Received: %v",
				b, expected[1], stream.KeysPayloadB.Get(b).Bytes())
		}

		if !reflect.DeepEqual(output.PartialPayloadACypherText, expected[2]) {
			t.Errorf("PermuteStream.Output() incorrect received CypherPayloadA data at %v: Expected: %v, Received: %v",
				b, expected[2], stream.CypherPayloadA.Get(b).Bytes())
		}

		if !reflect.DeepEqual(output.PartialPayloadBCypherText, expected[3]) {
			t.Errorf("PermuteStream.Output() incorrect received CypherPayloadB data at %v: Expected: %v, Received: %v",
				b, expected[3], stream.CypherPayloadB.Get(b).Bytes())
		}

	}

}

// Tests that PermuteStream conforms to the CommsStream interface
func TestPermuteStream_CommsInterface(t *testing.T) {

	var face interface{}
	face = &PermuteStream{}
	_, ok := face.(services.Stream)

	if !ok {
		t.Errorf("PermuteStream: Does not conform to the Stream interface")
	}

}

func TestPermuteGraph(t *testing.T) {
	grp := initPermuteGroup()

	batchSize := uint32(100)

	expectedName := "PrecompPermute"

	// Show that the Inti function meets the function type
	var graphInit graphs.Initializer
	graphInit = InitPermuteGraph

	PanicHandler := func(g, m string, err error) {
		panic(fmt.Sprintf("Error in module %s of graph %s: %s", g, m, err.Error()))
	}

	gc := services.NewGraphGenerator(4, uint8(runtime.NumCPU()), 1, 1.0)

	// Initialize graph
	g := graphInit(gc)

	if g.GetName() != expectedName {
		t.Errorf("PrecompPermute has incorrect name Expected %s, Received %s", expectedName, g.GetName())
	}

	// Build the graph
	g.Build(batchSize, PanicHandler)

	var done *uint32
	done = new(uint32)
	*done = 0

	// Build the roundBuffer
	roundBuffer := round.NewBuffer(grp, g.GetBatchSize(), g.GetExpandedBatchSize())

	subPermutation := roundBuffer.Permutations[:batchSize]

	shuffle.Shuffle32(&subPermutation)

	// Link the graph to the roundBuffer. building the stream object
	g.Link(grp, roundBuffer)

	permuteInverse := make([]uint32, g.GetBatchSize())
	for i := uint32(0); i < uint32(len(permuteInverse)); i++ {
		permuteInverse[roundBuffer.Permutations[i]] = i
	}

	stream := g.GetStream().(*PermuteStream)

	//fill the fields of the stream object for testing
	grp.Random(stream.PublicCypherKey)

	for i := uint32(0); i < g.GetExpandedBatchSize(); i++ {
		grp.Random(stream.S.Get(i))
		grp.Random(stream.V.Get(i))
		grp.Random(stream.Y_S.Get(i))
		grp.Random(stream.Y_V.Get(i))
	}

	// Build i/o used for testing
	KeysPayloadAExpected := grp.NewIntBuffer(g.GetExpandedBatchSize(), grp.NewInt(1))
	CypherPayloadAExpected := grp.NewIntBuffer(g.GetExpandedBatchSize(), grp.NewInt(1))
	KeysPayloadBExpected := grp.NewIntBuffer(g.GetExpandedBatchSize(), grp.NewInt(1))
	CypherPayloadBExpected := grp.NewIntBuffer(g.GetExpandedBatchSize(), grp.NewInt(1))

	for i := uint32(0); i < batchSize; i++ {
		grp.SetUint64(stream.KeysPayloadA.Get(i), uint64(i+1))
		grp.SetUint64(stream.CypherPayloadA.Get(i), uint64(i+11))
		grp.SetUint64(stream.KeysPayloadB.Get(i), uint64(i+111))
		grp.SetUint64(stream.CypherPayloadB.Get(i), uint64(i+1111))
	}

	for i := uint32(0); i < batchSize; i++ {

		grp.Set(KeysPayloadAExpected.Get(i), stream.KeysPayloadA.Get(i))
		grp.Set(CypherPayloadAExpected.Get(i), stream.CypherPayloadA.Get(i))
		grp.Set(KeysPayloadBExpected.Get(i), stream.KeysPayloadB.Get(i))
		grp.Set(CypherPayloadBExpected.Get(i), stream.CypherPayloadB.Get(i))

		s := stream

		// Compute expected result for this slot
		cryptops.ElGamal(grp, s.S.Get(i), s.Y_S.Get(i), s.PublicCypherKey, KeysPayloadAExpected.Get(i), CypherPayloadAExpected.Get(i))
		// Execute elgamal on the keys for the Associated Data
		cryptops.ElGamal(s.Grp, s.V.Get(i), s.Y_V.Get(i), s.PublicCypherKey, KeysPayloadBExpected.Get(i), CypherPayloadBExpected.Get(i))

	}

	g.Run()

	go func(g *services.Graph) {

		for i := uint32(0); i < g.GetBatchSize()-1; i++ {
			g.Send(services.NewChunk(i, i+1), nil)
		}

		atomic.AddUint32(done, 1)
		g.Send(services.NewChunk(g.GetExpandedBatchSize()-1, g.GetExpandedBatchSize()), nil)
	}(g)

	ok := true
	var chunk services.Chunk

	for ok {
		chunk, ok = g.GetOutput()
		for i := chunk.Begin(); i < chunk.End(); i++ {
			d := atomic.LoadUint32(done)

			if d == 0 {
				t.Error("Permute: should not be outputting until all inputs are inputted")
			}

			if stream.KeysPayloadAPermuted[i].Cmp(KeysPayloadAExpected.Get(permuteInverse[i])) != 0 {
				t.Error(fmt.Sprintf("Permute: KeysPayloadA slot %v out1 not permuted correctly", i))
			}

			if stream.CypherPayloadAPermuted[i].Cmp(CypherPayloadAExpected.Get(permuteInverse[i])) != 0 {
				t.Error(fmt.Sprintf("Permute: CypherPayloadA slot %v out1 not permuted correctly", i))
			}

			if stream.KeysPayloadBPermuted[i].Cmp(KeysPayloadBExpected.Get(permuteInverse[i])) != 0 {
				t.Error(fmt.Sprintf("Permute: KeysPayloadB slot %v out2 not permuted correctly", i))
			}

			if stream.CypherPayloadBPermuted[i].Cmp(CypherPayloadBExpected.Get(permuteInverse[i])) != 0 {
				t.Error(fmt.Sprintf("Permute: CypherPayloadB slot %v out2 not permuted correctly", i))
			}

		}
	}

}

func initPermuteGroup() *cyclic.Group {
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
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16), large.NewInt(2))
	return grp
}

func initPermuteRoundBuffer(grp *cyclic.Group, batchSize uint32) *round.Buffer {
	return round.NewBuffer(grp, batchSize, batchSize)

}
