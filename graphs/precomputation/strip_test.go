////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package precomputation

import (
	"fmt"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/gpumathsgo/cryptops"
	"gitlab.com/elixxir/server/graphs"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/xx_network/crypto/large"
	"reflect"
	"runtime"
	"testing"
)

// Test that StripStream.GetName() returns the correct name
func TestStripStream_GetName(t *testing.T) {
	expected := "PrecompStripStream"

	stream := StripStream{}

	if stream.GetName() != expected {
		t.Errorf("StripStream.GetName(), Expected %s, Received %s", expected, stream.GetName())
	}
}

// Test that StripStream.Link() Links correctly
func TestStripStream_Link(t *testing.T) {
	grp := initStripGroup()

	stream := StripStream{}

	batchSize := uint32(100)

	roundBuffer := initStripRoundBuffer(grp, batchSize)

	stream.Link(grp, batchSize, roundBuffer)

	if roundBuffer.Z.Cmp(stream.Z) != 0 {
		t.Errorf(
			"RevealStream.Link() Z value not linked: Expected %s, Received %s",
			roundBuffer.Z.TextVerbose(10, 16), stream.Z.TextVerbose(10, 16))
	}

	checkStreamIntBuffer(grp, stream.PayloadAPrecomputation, roundBuffer.PayloadAPrecomputation, "PayloadAPrecomputation", t)
	checkStreamIntBuffer(grp, stream.PayloadBPrecomputation, roundBuffer.PayloadBPrecomputation, "PayloadBPrecomputation", t)

	checkIntBuffer(stream.CypherPayloadA, batchSize, "CypherPayloadA", grp.NewInt(1), t)
	checkIntBuffer(stream.CypherPayloadB, batchSize, "CypherPayloadB", grp.NewInt(1), t)

	// Edit round to show that Z value in stream changes
	expected := grp.Random(roundBuffer.Z)

	if stream.Z.Cmp(expected) != 0 {
		t.Errorf(
			"StripStream.Link() Z value not linked to round: Expected %s, Received %s",
			roundBuffer.Z.TextVerbose(10, 16), stream.Z.TextVerbose(10, 16))
	}
}

// Tests Input's happy path
func TestStripStream_Input(t *testing.T) {
	grp := initStripGroup()

	stream := StripStream{}

	batchSize := uint32(100)

	roundBuffer := initStripRoundBuffer(grp, batchSize)

	stream.Link(grp, batchSize, roundBuffer)

	for b := uint32(0); b < batchSize; b++ {

		expected := [][]byte{
			{byte(1)},
			{byte(1)},
		}

		msg := &mixmessages.Slot{
			PartialPayloadACypherText: expected[0],
			PartialPayloadBCypherText: expected[1],
		}

		err := stream.Input(b, msg)
		if err != nil {
			t.Errorf("StripStream.Input() errored on slot %v: %s", b, err.Error())
		}

		if !reflect.DeepEqual(stream.CypherPayloadA.Get(b).Bytes(), expected[0]) {
			t.Errorf("StripStream.Input() incorrect stored CypherPayloadA data at %v: Expected: %v, Received: %v",
				b, expected[0], stream.CypherPayloadA.Get(b).Bytes())
		}

		if !reflect.DeepEqual(stream.CypherPayloadB.Get(b).Bytes(), expected[1]) {
			t.Errorf("StripStream.Input() incorrect stored CypherPayloadB data at %v: Expected: %v, Received: %v",
				b, expected[1], stream.CypherPayloadB.Get(b).Bytes())
		}

	}

}

// Tests that the input errors correctly when the index is outside of the batch
func TestStripStream_Input_OutOfBatch(t *testing.T) {
	grp := initStripGroup()

	stream := StripStream{}

	batchSize := uint32(100)

	roundBuffer := initStripRoundBuffer(grp, batchSize)

	stream.Link(grp, batchSize, roundBuffer)
	msg := &mixmessages.Slot{
		PartialPayloadACypherText: []byte{0},
		PartialPayloadBCypherText: []byte{0},
	}

	err := stream.Input(batchSize, msg)

	if err != services.ErrOutsideOfBatch {
		t.Errorf("StripStream.Input() did nto return an outside of batch error when out of batch")
	}

	err1 := stream.Input(batchSize+1, msg)

	if err1 != services.ErrOutsideOfBatch {
		t.Errorf("StripStream.Input() did not return an outside of batch error when out of batch")
	}
}

// Tests that Input errors correct when the passed value is out of the group
func TestStripStream_Input_OutOfGroup(t *testing.T) {
	grp := cyclic.NewGroup(large.NewInt(11), large.NewInt(4))

	stream := StripStream{}

	batchSize := uint32(100)

	roundBuffer := initStripRoundBuffer(grp, batchSize)

	stream.Link(grp, batchSize, roundBuffer)

	msg := &mixmessages.Slot{
		PartialPayloadACypherText: large.NewInt(89).Bytes(),
		PartialPayloadBCypherText: large.NewInt(13).Bytes(),
	}

	err := stream.Input(batchSize-10, msg)

	if err != services.ErrOutsideOfGroup {
		t.Errorf("StripStream.Input() did not return an error when out of group")
	}
}

// Tests that the output function returns a valid cmixMessage
func TestStripStream_Output(t *testing.T) {
	grp := initStripGroup()

	stream := StripStream{}

	batchSize := uint32(100)

	roundBuffer := initStripRoundBuffer(grp, batchSize)

	stream.Link(grp, batchSize, roundBuffer)

	for b := uint32(0); b < batchSize; b++ {

		expected := [][]byte{
			{byte(b + 1), 0},
			{byte(b + 1), 1},
		}

		grp.SetBytes(stream.PayloadAPrecomputation.Get(b), expected[0])
		grp.SetBytes(stream.PayloadBPrecomputation.Get(b), expected[1])

		output := stream.Output(b)

		if !reflect.DeepEqual(output.EncryptedPayloadAKeys, expected[0]) {
			t.Errorf("StripStream.Output() incorrect received "+
				"CypherPayloadA data at %v: Expected: %v, Received: %v",
				b, expected[0], output.EncryptedPayloadAKeys)
		}

		if !reflect.DeepEqual(output.EncryptedPayloadBKeys, expected[1]) {
			t.Errorf("StripStream.Output() incorrect received CypherPayloadB"+
				" data at %v: Expected: %v, Received: %v",
				b, expected[1], output.EncryptedPayloadBKeys)
		}

	}

}

// Tests that StripStream conforms to the CommsStream interface
func TestStripStream_Interface(t *testing.T) {

	var face interface{}
	face = &StripStream{}
	_, ok := face.(services.Stream)

	if !ok {
		t.Errorf("StripStream: Does not conform to the CommsStream interface")
	}

}

func TestStrip_Graph(t *testing.T) {
	viper.Set("useGPU", false)
	grp := initStripGroup()

	batchSize := uint32(100)

	// Show that the Init function meets the function type
	var graphInit graphs.Initializer
	graphInit = InitStripGraph

	PanicHandler := func(g, m string, err error) {
		panic(fmt.Sprintf("Error in module %s of graph %s: %s", g, m, err.Error()))
	}

	gc := services.NewGraphGenerator(4, uint8(runtime.NumCPU()), services.AutoOutputSize, 0)

	// Initialize graph
	g := graphInit(gc)

	// Build the graph
	g.Build(batchSize, PanicHandler)

	// Build the round
	roundBuffer := round.NewBuffer(grp, g.GetBatchSize(), g.GetExpandedBatchSize())
	roundBuffer.InitLastNode()

	for i := uint32(0); i < g.GetExpandedBatchSize(); i++ {
		roundBuffer.PermutedPayloadAKeys[i] = grp.NewInt(1)
		roundBuffer.PermutedPayloadBKeys[i] = grp.NewInt(1)
	}

	// Fill the fields of the round object for testing
	for i := uint32(0); i < g.GetBatchSize(); i++ {
		grp.Set(roundBuffer.PayloadBPrecomputation.Get(i), grp.NewInt(int64(1)))
		grp.Set(roundBuffer.PayloadAPrecomputation.Get(i), grp.NewInt(int64(1)))
	}

	grp.FindSmallCoprimeInverse(roundBuffer.Z, 256)

	// Link the graph to the round. building the stream object
	g.Link(grp, roundBuffer)

	stream := g.GetStream().(*StripStream)

	CypherPayloadAExpected := grp.NewIntBuffer(g.GetExpandedBatchSize(), grp.NewInt(1))
	CypherPayloadBExpected := grp.NewIntBuffer(g.GetExpandedBatchSize(), grp.NewInt(1))

	// Fill the fields of the stream object for testing
	for i := uint32(0); i < g.GetBatchSize(); i++ {
		grp.RandomCoprime(stream.CypherPayloadA.Get(i))
		grp.RandomCoprime(stream.CypherPayloadB.Get(i))

		//These two lines copy the generated values
		grp.Set(CypherPayloadAExpected.Get(i), stream.CypherPayloadA.Get(i))
		grp.Set(CypherPayloadBExpected.Get(i), stream.CypherPayloadB.Get(i))

	}

	// Build i/o used for testing
	PayloadAPrecomputationExpected := grp.NewIntBuffer(g.GetExpandedBatchSize(), grp.NewInt(1))
	PayloadBPrecomputationExpected := grp.NewIntBuffer(g.GetExpandedBatchSize(), grp.NewInt(1))

	// Run the graph
	g.Run()

	// Send inputs into the graph
	go func(g *services.Graph) {
		for i := uint32(0); i < g.GetBatchSize(); i++ {
			g.Send(services.NewChunk(i, i+1), nil)
		}
	}(g)

	// Get the output
	s := g.GetStream().(*StripStream)

	ok := true
	var chunk services.Chunk

	for ok {
		chunk, ok = g.GetOutput()
		tmp := s.Grp.NewInt(1)
		for i := chunk.Begin(); i < chunk.End(); i++ {
			// Compute expected root coprime for both payloads
			cryptops.RootCoprime(s.Grp, CypherPayloadAExpected.Get(i), s.Z, tmp)
			s.Grp.Set(CypherPayloadAExpected.Get(i), tmp)

			cryptops.RootCoprime(s.Grp, CypherPayloadBExpected.Get(i), s.Z, tmp)
			s.Grp.Set(CypherPayloadBExpected.Get(i), tmp)

			// Compute inverse
			cryptops.Inverse(s.Grp, PayloadAPrecomputationExpected.Get(i), PayloadAPrecomputationExpected.Get(i))
			cryptops.Inverse(s.Grp, PayloadBPrecomputationExpected.Get(i), PayloadBPrecomputationExpected.Get(i))

			// Compute mul2
			cryptops.Mul2(s.Grp, s.CypherPayloadA.Get(i), PayloadAPrecomputationExpected.Get(i))
			cryptops.Mul2(s.Grp, s.CypherPayloadB.Get(i), PayloadBPrecomputationExpected.Get(i))

			// Verify payloads match the expected values
			if PayloadAPrecomputationExpected.Get(i).Cmp(s.PayloadAPrecomputation.Get(i)) != 0 {
				t.Error(fmt.Sprintf("PrecompStrip: PayloadA Keys Cypher not equal on slot %v expected %v received %v",
					i, PayloadAPrecomputationExpected.Get(i).Text(16), s.PayloadAPrecomputation.Get(i).Text(16)))
			}

			if PayloadBPrecomputationExpected.Get(i).Cmp(s.PayloadBPrecomputation.Get(i)) != 0 {
				t.Error(fmt.Sprintf("PrecompStrip: PayloadB Keys Cypher not equal on slot %v expected %v received %v",
					i, PayloadBPrecomputationExpected.Get(i).Text(16), s.PayloadBPrecomputation.Get(i).Text(16)))
			}
		}
	}
}

func initStripGroup() *cyclic.Group {
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

func initStripRoundBuffer(grp *cyclic.Group, batchSize uint32) *round.Buffer {
	return round.NewBuffer(grp, batchSize, batchSize)
}
