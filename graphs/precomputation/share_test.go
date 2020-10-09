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
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/server/graphs"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/services"
	"reflect"
	"runtime"
	"testing"
)

// Test that RevealStream.GetName() returns the correct name
func TestShareStream_GetName(t *testing.T) {
	expected := "PrecompShareStream"

	rs := ShareStream{}

	if rs.GetName() != expected {
		t.Errorf("ShareStream.GetName(), Expected %s, Received %s", expected, rs.GetName())
	}
}

// Test that RevealStream.Link() Links correctly
func TestShareStream_Link(t *testing.T) {
	grp := initShareGroup()

	stream := ShareStream{}

	batchSize := uint32(100)

	roundBuffer := initShareRoundBuffer(grp, batchSize)

	stream.Link(grp, batchSize, roundBuffer)

	if roundBuffer.Z.Cmp(stream.Z) != 0 {
		t.Errorf(
			"ShareStream.Link() Z value not linked: Expected %s, Received %s",
			roundBuffer.Z.TextVerbose(10, 16), stream.Z.TextVerbose(10, 16))
	}

	// Edit round to show that Z value in stream changes
	expected := grp.Random(roundBuffer.Z)

	if stream.Z.Cmp(expected) != 0 {
		t.Errorf(
			"RevealStream.Link() Z value not linked to round: Expected %s, Received %s",
			roundBuffer.Z.TextVerbose(10, 16), stream.Z.TextVerbose(10, 16))
	}

	if stream.PartialPublicCypherKey == nil {
		t.Errorf(
			"ShareStream.Link(): Partial Cypher Key not set")
	}
}

// Tests Input's happy path
func TestShareStream_Input(t *testing.T) {
	grp := initShareGroup()

	stream := ShareStream{}

	batchSize := uint32(100)

	roundBuffer := initShareRoundBuffer(grp, batchSize)

	stream.Link(grp, batchSize, roundBuffer)

	msg := &mixmessages.Slot{
		PartialRoundPublicCypherKey: []byte{1},
	}

	err := stream.Input(0, msg)

	if err != nil {
		t.Errorf("RevealStream.Input() errored: %s", err.Error())
	}

	if stream.PartialPublicCypherKey.Cmp(grp.NewInt(1)) != 0 {
		t.Errorf("ShareStream.Input() partialPublicCypherKey not set to 1: %s",
			stream.PartialPublicCypherKey.Text(16))
	}
}

// Tests that Input errors correct when the passed value is out of the group
func TestShareStream_Input_OutOfGroup(t *testing.T) {
	grp := cyclic.NewGroup(large.NewInt(11), large.NewInt(4))

	stream := ShareStream{}

	batchSize := uint32(100)

	roundBuffer := initShareRoundBuffer(grp, batchSize)

	stream.Link(grp, batchSize, roundBuffer)

	msg := &mixmessages.Slot{
		PartialRoundPublicCypherKey: []byte{0},
	}

	err := stream.Input(0, msg)

	if err != services.ErrOutsideOfGroup {
		t.Errorf("SharetStream.Input() did not return an error when out of group")
	}
}

// Tests that the output function returns a valid cmixMessage
func TestShareStream_Output(t *testing.T) {
	grp := initShareGroup()

	stream := ShareStream{}

	batchSize := uint32(100)

	roundBuffer := initShareRoundBuffer(grp, batchSize)

	stream.Link(grp, batchSize, roundBuffer)

	expected := []byte{1}

	msg := &mixmessages.Slot{
		PartialRoundPublicCypherKey: expected,
	}

	err := stream.Input(0, msg)

	if err != nil {
		t.Errorf("RevealStream.Output() errored on input: %s", err.Error())
	}

	output := stream.Output(0)

	if !reflect.DeepEqual(output.PartialRoundPublicCypherKey, expected) {
		t.Errorf("RevealStream.Output() incorrect received Partial Round Cypher Key: Expected: %v, Received: %v",
			expected, output.PartialRoundPublicCypherKey)
	}

}

// Tests that RevealStream conforms to the CommsStream interface
func TestShareStream_Interface(t *testing.T) {

	var face interface{}
	face = &ShareStream{}
	_, ok := face.(services.Stream)

	if !ok {
		t.Errorf("RevealStream: Does not conform to the CommsStream interface")
	}

}

func TestShare_Graph(t *testing.T) {
	grp := initShareGroup()

	batchSize := uint32(1)

	// Show that the Init function meets the function type
	var graphInit graphs.Initializer
	graphInit = InitShareGraph

	PanicHandler := func(g, m string, err error) {
		panic(fmt.Sprintf("Error in module %s of graph %s: %s", g, m, err.Error()))
	}

	gc := services.NewGraphGenerator(1, uint8(runtime.NumCPU()), services.AutoOutputSize, 0)

	//Initialize graph
	g := graphInit(gc)

	// Build the graph
	g.Build(batchSize, PanicHandler)

	// Build the round
	roundBuffer := round.NewBuffer(grp, g.GetBatchSize(), g.GetExpandedBatchSize())

	// Fill the fields of the stream object for testing
	grp.FindSmallCoprimeInverse(roundBuffer.Z, 256)

	// Link the graph to the round. building the stream object
	g.Link(grp, roundBuffer)

	stream := g.GetStream().(*ShareStream)
	grp.SetUint64(stream.PartialPublicCypherKey, 2)

	// Build i/o used for testing
	PubicCypherKeyExpected := grp.ExpG(roundBuffer.Z, grp.NewInt(1))

	// Run the graph
	g.Run()

	// Send inputs into the graph
	go func(g *services.Graph) {

		g.Send(services.NewChunk(0, 1), nil)
	}(g)

	ok := true
	var chunk services.Chunk

	for ok {
		chunk, ok = g.GetOutput()
		for i := chunk.Begin(); i < chunk.End(); i++ {
			if PubicCypherKeyExpected.Cmp(stream.PartialPublicCypherKey) != 0 {
				t.Errorf("PrecompShare:PartialPublicCypherKey incorrect, Expected: %v, Received: %v",
					PubicCypherKeyExpected.Text(16), stream.PartialPublicCypherKey.Text(16))
			}
		}
	}
}

func initShareGroup() *cyclic.Group {
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

func initShareRoundBuffer(grp *cyclic.Group, batchSize uint32) *round.Buffer {
	return round.NewBuffer(grp, batchSize, batchSize)
}
