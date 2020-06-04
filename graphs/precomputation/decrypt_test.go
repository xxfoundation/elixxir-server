////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package precomputation

import (
	"fmt"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/server/graphs"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/services"
	"reflect"
	"runtime"
	"testing"
)

// Test that DecryptStream.GetName() returns the correct name
func TestDecryptStream_GetName(t *testing.T) {
	expected := "PrecompDecryptStream"

	ds := DecryptStream{}

	if ds.GetName() != expected {
		t.Errorf("DecryptStream.GetName(), Expected %s, Recieved %s", expected, ds.GetName())
	}
}

// Test that DecryptStream.Link() Links correctly
func TestDecryptStream_Link(t *testing.T) {
	grp := initDecryptGroup()

	stream := DecryptStream{}

	batchSize := uint32(100)

	roundBuffer := initDecryptRoundBuffer(grp, batchSize)

	stream.Link(grp, batchSize, roundBuffer)

	checkStreamIntBuffer(grp, stream.R, roundBuffer.R, "R", t)
	checkStreamIntBuffer(grp, stream.U, roundBuffer.U, "U", t)
	checkStreamIntBuffer(grp, stream.Y_R, roundBuffer.Y_R, "Y_R", t)
	checkStreamIntBuffer(grp, stream.Y_U, roundBuffer.Y_U, "Y_U", t)

	checkIntBuffer(stream.KeysPayloadA, batchSize, "KeysPayloadA", grp.NewInt(1), t)
	checkIntBuffer(stream.CypherPayloadA, batchSize, "CypherPayloadA", grp.NewInt(1), t)
	checkIntBuffer(stream.KeysPayloadB, batchSize, "KeysPayloadB", grp.NewInt(1), t)
	checkIntBuffer(stream.CypherPayloadB, batchSize, "CypherPayloadB", grp.NewInt(1), t)
}

// Test Input's happy path
func TestDecryptStream_Input(t *testing.T) {
	grp := initDecryptGroup()

	stream := DecryptStream{}

	batchSize := uint32(100)

	roundBuffer := initDecryptRoundBuffer(grp, batchSize)

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
			t.Errorf("DecryptStream.Input() errored on slot %v: %s", b, err.Error())
		}

		if !reflect.DeepEqual(stream.KeysPayloadA.Get(b).Bytes(), expected[0]) {
			t.Errorf("DecryptStream.Input() incorrect stored KeysPayloadA data at %v: Expected: %v, Recieved: %v",
				b, expected[0], stream.KeysPayloadA.Get(b).Bytes())
		}

		if !reflect.DeepEqual(stream.KeysPayloadB.Get(b).Bytes(), expected[1]) {
			t.Errorf("DecryptStream.Input() incorrect stored KeysPayloadB data at %v: Expected: %v, Recieved: %v",
				b, expected[1], stream.KeysPayloadB.Get(b).Bytes())
		}

		if !reflect.DeepEqual(stream.CypherPayloadA.Get(b).Bytes(), expected[2]) {
			t.Errorf("DecryptStream.Input() incorrect stored CypherPayloadA data at %v: Expected: %v, Recieved: %v",
				b, expected[2], stream.CypherPayloadA.Get(b).Bytes())
		}

		if !reflect.DeepEqual(stream.CypherPayloadB.Get(b).Bytes(), expected[3]) {
			t.Errorf("DecryptStream.Input() incorrect stored CypherPayloadB data at %v: Expected: %v, Recieved: %v",
				b, expected[3], stream.CypherPayloadB.Get(b).Bytes())
		}

	}

}

// Tests that the input errors correctly when the index is outside of the batch
func TestDecryptStream_Input_OutOfBatch(t *testing.T) {
	grp := initDecryptGroup()

	stream := DecryptStream{}

	batchSize := uint32(100)

	roundBuffer := initDecryptRoundBuffer(grp, batchSize)

	stream.Link(grp, batchSize, roundBuffer)

	msg := &mixmessages.Slot{
		EncryptedPayloadAKeys:     []byte{0},
		EncryptedPayloadBKeys:     []byte{0},
		PartialPayloadACypherText: []byte{0},
		PartialPayloadBCypherText: []byte{0},
	}

	err := stream.Input(batchSize, msg)

	if err == nil {
		t.Errorf("DecryptStream.Input() did nto return an error when out of batch")
	}

	err1 := stream.Input(batchSize+1, msg)

	if err1 == nil {
		t.Errorf("DecryptStream.Input() did nto return an error when out of batch")
	}
}

// Tests that Input errors correct when the passed value is out of the group
func TestDecryptStream_Input_OutOfGroup(t *testing.T) {
	grp := initDecryptGroup()

	stream := DecryptStream{}

	batchSize := uint32(100)

	roundBuffer := initDecryptRoundBuffer(grp, batchSize)

	stream.Link(grp, batchSize, roundBuffer)

	msg := &mixmessages.Slot{
		EncryptedPayloadAKeys:     []byte{0},
		EncryptedPayloadBKeys:     []byte{0},
		PartialPayloadACypherText: []byte{0},
		PartialPayloadBCypherText: []byte{0},
	}

	err := stream.Input(batchSize-10, msg)

	if err != services.ErrOutsideOfGroup {
		t.Errorf("DecryptStream.Input() did not return an error when out of group")
	}
}

// Tests that the output function returns a valid cmixMessage
func TestDecryptStream_Output(t *testing.T) {
	grp := initDecryptGroup()

	stream := DecryptStream{}

	batchSize := uint32(100)

	roundBuffer := initDecryptRoundBuffer(grp, batchSize)

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
			t.Errorf("DecryptStream.Output() errored on slot %v: %s", b, err.Error())
		}

		output := stream.Output(b)

		if !reflect.DeepEqual(output.EncryptedPayloadAKeys, expected[0]) {
			t.Errorf("DecryptStream.Output() incorrect recieved KeysPayloadA data at %v: Expected: %v, Recieved: %v",
				b, expected[0], stream.KeysPayloadA.Get(b).Bytes())
		}

		if !reflect.DeepEqual(output.EncryptedPayloadBKeys, expected[1]) {
			t.Errorf("DecryptStream.Output() incorrect recieved KeysPayloadB data at %v: Expected: %v, Recieved: %v",
				b, expected[1], stream.KeysPayloadB.Get(b).Bytes())
		}

		if !reflect.DeepEqual(output.PartialPayloadACypherText, expected[2]) {
			t.Errorf("DecryptStream.Output() incorrect recieved CypherPayloadA data at %v: Expected: %v, Recieved: %v",
				b, expected[2], stream.CypherPayloadA.Get(b).Bytes())
		}

		if !reflect.DeepEqual(output.PartialPayloadBCypherText, expected[3]) {
			t.Errorf("DecryptStream.Output() incorrect recieved CypherPayloadB data at %v: Expected: %v, Recieved: %v",
				b, expected[3], stream.CypherPayloadB.Get(b).Bytes())
		}

	}

}

//Tests that DecryptStream conforms to the CommsStream interface
func TestDecryptStream_Interface(t *testing.T) {

	var face interface{}
	face = &DecryptStream{}
	_, ok := face.(services.Stream)

	if !ok {
		t.Errorf("DecryptStream: Does not conform to the Stream interface")
	}

}

func TestDecryptGraph(t *testing.T) {

	grp := initDecryptGroup()

	batchSize := uint32(100)

	expectedName := "PrecompDecrypt"

	//Show that the Inti function meets the function type
	var graphInit graphs.Initializer
	graphInit = InitDecryptGraph

	PanicHandler := func(g, m string, err error) {
		panic(fmt.Sprintf("Error in module %s of graph %s: %s", g, m, err.Error()))
	}

	gc := services.NewGraphGenerator(4, uint8(runtime.NumCPU()), services.AutoOutputSize, 1.0)

	//Initialize graph
	g := graphInit(gc)

	if g.GetName() != expectedName {
		t.Errorf("PrecompDecrypt has incorrect name Expected %s, Recieved %s", expectedName, g.GetName())
	}

	//Build the graph
	g.Build(batchSize, PanicHandler)

	//Build the round
	roundBuffer := round.NewBuffer(grp, g.GetBatchSize(), g.GetExpandedBatchSize())

	//Link the graph to the round. building the stream object
	g.Link(grp, roundBuffer)

	stream := g.GetStream().(*DecryptStream)

	//fill the fields of the stream object for testing
	grp.Random(stream.PublicCypherKey)

	for i := uint32(0); i < g.GetExpandedBatchSize(); i++ {
		grp.Random(stream.R.Get(i))
		grp.Random(stream.U.Get(i))
		grp.Random(stream.Y_R.Get(i))
		grp.Random(stream.Y_U.Get(i))
	}

	//Build i/o used for testing
	KeysPayloadAExpected := grp.NewIntBuffer(g.GetExpandedBatchSize(), grp.NewInt(1))
	CypherPayloadAExpected := grp.NewIntBuffer(g.GetExpandedBatchSize(), grp.NewInt(1))
	KeysPayloadBExpected := grp.NewIntBuffer(g.GetExpandedBatchSize(), grp.NewInt(1))
	CypherPayloadBExpected := grp.NewIntBuffer(g.GetExpandedBatchSize(), grp.NewInt(1))

	//Run the graph
	g.Run()

	//Send inputs into the graph
	go func(g *services.Graph) {
		for i := uint32(0); i < g.GetBatchSize(); i++ {
			g.Send(services.NewChunk(i, i+1), nil)
		}
	}(g)

	//Get the output
	s := g.GetStream().(*DecryptStream)

	ok := true
	var chunk services.Chunk

	for ok {
		chunk, ok = g.GetOutput()
		for i := chunk.Begin(); i < chunk.End(); i++ {
			// Compute expected result for this slot
			cryptops.ElGamal(s.Grp, s.R.Get(i), s.Y_R.Get(i), s.PublicCypherKey, KeysPayloadAExpected.Get(i), CypherPayloadAExpected.Get(i))
			//Execute elgamal on the keys for the Associated Data
			cryptops.ElGamal(s.Grp, s.U.Get(i), s.Y_U.Get(i), s.PublicCypherKey, KeysPayloadBExpected.Get(i), CypherPayloadBExpected.Get(i))

			if KeysPayloadAExpected.Get(i).Cmp(s.KeysPayloadA.Get(i)) != 0 {
				t.Error(fmt.Sprintf("PrecompDecrypt: PayloadA Keys not equal on slot %v", i))
			}
			if CypherPayloadAExpected.Get(i).Cmp(s.CypherPayloadA.Get(i)) != 0 {
				t.Error(fmt.Sprintf("PrecompDecrypt: PayloadA Keys Cypher not equal on slot %v", i))
			}
			if KeysPayloadBExpected.Get(i).Cmp(s.KeysPayloadB.Get(i)) != 0 {
				t.Error(fmt.Sprintf("PrecompDecrypt: PayloadB Keys not equal on slot %v", i))
			}
			if CypherPayloadBExpected.Get(i).Cmp(s.CypherPayloadB.Get(i)) != 0 {
				t.Error(fmt.Sprintf("PrecompDecrypt: PayloadB Keys Cypher not equal on slot %v", i))
			}
		}
	}
}

func checkStreamIntBuffer(grp *cyclic.Group, ib, sourceib *cyclic.IntBuffer, source string, t *testing.T) {
	if ib.Len() != sourceib.Len() {
		t.Errorf("preomp.DecryptStream.Link: Length of intBuffer %s not correct, "+
			"Expected %v, Recieved: %v", source, sourceib.Len(), ib.Len())
	}

	numBad := 0
	for i := 0; i < sourceib.Len(); i++ {
		grp.SetUint64(sourceib.Get(uint32(i)), uint64(i))
		ci := ib.Get(uint32(i))
		if ci.Cmp(sourceib.Get(uint32(i))) != 0 {
			numBad++
		}
	}

	if numBad != 0 {
		t.Errorf("preomp.DecryptStream.Link: Ints in %v/%v intBuffer %s intilized incorrectly",
			numBad, sourceib.Len(), source)
	}
}

func checkIntBuffer(ib *cyclic.IntBuffer, expandedBatchSize uint32, source string, defaultInt *cyclic.Int, t *testing.T) {
	if ib.Len() != int(expandedBatchSize) {
		t.Errorf("New RoundBuffer: Length of intBuffer %s not correct, "+
			"Expected %v, Recieved: %v", source, expandedBatchSize, ib.Len())
	}

	numBad := 0
	for i := uint32(0); i < expandedBatchSize; i++ {
		ci := ib.Get(i)
		if ci.Cmp(defaultInt) != 0 {
			numBad++
		}
	}

	if numBad != 0 {
		t.Errorf("New RoundBuffer: Ints in %v/%v intBuffer %s intilized incorrectly",
			numBad, expandedBatchSize, source)
	}
}

func initDecryptGroup() *cyclic.Group {
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

func initDecryptRoundBuffer(grp *cyclic.Group, batchSize uint32) *round.Buffer {
	return round.NewBuffer(grp, batchSize, batchSize)
}
