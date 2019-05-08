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
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/services"
	"reflect"
	"runtime"
	"testing"
)

// Test that RevealStream.GetName() returns the correct name
func TestRevealStream_GetName(t *testing.T) {
	expected := "PrecompRevealStream"

	rs := RevealStream{}

	if rs.GetName() != expected {
		t.Errorf("RevealStream.GetName(), Expected %s, Recieved %s", expected, rs.GetName())
	}
}

// Test that RevealStream.Link() Links correctly
func TestRevealStream_Link(t *testing.T) {
	grp := initRevealGroup()

	stream := RevealStream{}

	batchSize := uint32(100)

	roundBuffer := initRevealRoundBuffer(grp, batchSize)

	stream.Link(grp, batchSize, roundBuffer)

	if roundBuffer.Z.Cmp(stream.Z) != 0 {
		t.Errorf(
			"RevealStream.Link() Z value not linked: Expected %s, Recieved %s",
			roundBuffer.Z.TextVerbose(10, 16), stream.Z.TextVerbose(10, 16))
	}

	checkIntBuffer(stream.CypherMsg, batchSize, "CypherMsg", grp.NewInt(1), t)
	checkIntBuffer(stream.CypherAD, batchSize, "CypherAD", grp.NewInt(1), t)

	// Edit roundBuffer to show that Z value in stream changes
	expected := grp.Random(roundBuffer.Z)

	if stream.Z.Cmp(expected) != 0 {
		t.Errorf(
			"RevealStream.Link() Z value not linked to roundBuffer: Expected %s, Recieved %s",
			roundBuffer.Z.TextVerbose(10, 16), stream.Z.TextVerbose(10, 16))
	}
}

// Tests Input's happy path
func TestRevealtStream_Input(t *testing.T) {
	grp := initRevealGroup()

	stream := RevealStream{}

	batchSize := uint32(100)

	roundBuffer := initRevealRoundBuffer(grp, batchSize)

	stream.Link(grp, batchSize, roundBuffer)

	for b := uint32(0); b < batchSize; b++ {

		expected := [][]byte{
			{byte(b + 1), 0},
			{byte(b + 1), 1},
		}

		msg := &mixmessages.Slot{
			PartialMessageCypherText:        expected[0],
			PartialAssociatedDataCypherText: expected[1],
		}

		err := stream.Input(b, msg)
		if err != nil {
			t.Errorf("RevealStream.Input() errored on slot %v: %s", b, err.Error())
		}

		if !reflect.DeepEqual(stream.CypherMsg.Get(b).Bytes(), expected[0]) {
			t.Errorf("RevealStream.Input() incorrect stored CypherMsg data at %v: Expected: %v, Recieved: %v",
				b, expected[0], stream.CypherMsg.Get(b).Bytes())
		}

		if !reflect.DeepEqual(stream.CypherAD.Get(b).Bytes(), expected[1]) {
			t.Errorf("RevealStream.Input() incorrect stored CypherAD data at %v: Expected: %v, Recieved: %v",
				b, expected[1], stream.CypherAD.Get(b).Bytes())
		}

	}

}

// Tests that the input errors correctly when the index is outside of the batch
func TestRevealStream_Input_OutOfBatch(t *testing.T) {
	grp := initRevealGroup()

	stream := RevealStream{}

	batchSize := uint32(100)

	roundBuffer := initRevealRoundBuffer(grp, batchSize)

	stream.Link(grp, batchSize, roundBuffer)

	msg := &mixmessages.Slot{
		PartialMessageCypherText:        []byte{0},
		PartialAssociatedDataCypherText: []byte{0},
	}

	err := stream.Input(batchSize, msg)

	if err != services.ErrOutsideOfBatch {
		t.Errorf("RevealtStream.Input() did nto return an outside of batch error when out of batch")
	}

	err1 := stream.Input(batchSize+1, msg)

	if err1 != services.ErrOutsideOfBatch {
		t.Errorf("RevealStream.Input() did not return an outside of batch error when out of batch")
	}
}

// Tests that Input errors correct when the passed value is out of the group
func TestRevealStream_Input_OutOfGroup(t *testing.T) {
	grp := cyclic.NewGroup(large.NewInt(11), large.NewInt(4), large.NewInt(5))

	stream := RevealStream{}

	batchSize := uint32(100)

	roundBuffer := initRevealRoundBuffer(grp, batchSize)

	stream.Link(grp, batchSize, roundBuffer)

	msg := &mixmessages.Slot{
		PartialMessageCypherText:        large.NewInt(89).Bytes(),
		PartialAssociatedDataCypherText: large.NewInt(13).Bytes(),
	}

	err := stream.Input(batchSize-10, msg)

	if err != services.ErrOutsideOfGroup {
		t.Errorf("RevealStream.Input() did not return an error when out of group")
	}
}

// Tests that the output function returns a valid cmixMessage
func TestRevealStream_Output(t *testing.T) {
	grp := initRevealGroup()

	stream := RevealStream{}

	batchSize := uint32(100)

	roundBuffer := initRevealRoundBuffer(grp, batchSize)

	stream.Link(grp, batchSize, roundBuffer)

	for b := uint32(0); b < batchSize; b++ {

		expected := [][]byte{
			{byte(b + 1), 0},
			{byte(b + 1), 1},
		}

		msg := &mixmessages.Slot{
			PartialMessageCypherText:        expected[0],
			PartialAssociatedDataCypherText: expected[1],
		}

		err := stream.Input(b, msg)
		if err != nil {
			t.Errorf("RevealStream.Output() errored on slot %v: %s", b, err.Error())
		}

		output := stream.Output(b)

		if !reflect.DeepEqual(output.PartialMessageCypherText, expected[0]) {
			t.Errorf("RevealStream.Output() incorrect recieved CypherMsg data at %v: Expected: %v, Recieved: %v",
				b, expected[2], stream.CypherMsg.Get(b).Bytes())
		}

		if !reflect.DeepEqual(output.PartialAssociatedDataCypherText, expected[1]) {
			t.Errorf("RevealStream.Output() incorrect recieved CypherAD data at %v: Expected: %v, Recieved: %v",
				b, expected[3], stream.CypherAD.Get(b).Bytes())
		}

	}

}

// Tests that RevealStream conforms to the CommsStream interface
func TestRevealStream_Interface(t *testing.T) {

	var face interface{}
	face = &RevealStream{}
	_, ok := face.(services.Stream)

	if !ok {
		t.Errorf("RevealStream: Does not conform to the CommsStream interface")
	}

}

func TestReveal_Graph(t *testing.T) {

	grp := initRevealGroup()

	batchSize := uint32(100)

	// Show that the Init function meets the function type
	var graphInit graphs.Initializer
	graphInit = InitRevealGraph

	PanicHandler := func(err error) {
		panic(fmt.Sprintf("Reveal: Error in adapter: %s", err.Error()))
	}

	gc := services.NewGraphGenerator(4, PanicHandler, uint8(runtime.NumCPU()), services.AutoOutputSize, 0)

	//Initialize graph
	g := graphInit(gc)

	// Build the graph
	g.Build(batchSize)

	// Build the round
	roundBuffer := round.NewBuffer(grp, g.GetBatchSize(), g.GetExpandedBatchSize())

	// Fill the fields of the stream object for testing
	grp.FindSmallCoprimeInverse(roundBuffer.Z, 256)

	// Link the graph to the round. building the stream object
	g.Link(grp, roundBuffer)

	stream := g.GetStream().(*RevealStream)

	for i := uint32(0); i < g.GetExpandedBatchSize(); i++ {
		grp.RandomCoprime(stream.CypherMsg.Get(i))
		grp.RandomCoprime(stream.CypherAD.Get(i))
	}

	// Build i/o used for testing
	CypherMsgExpected := grp.NewIntBuffer(g.GetExpandedBatchSize(), grp.NewInt(1))
	CypherADExpected := grp.NewIntBuffer(g.GetExpandedBatchSize(), grp.NewInt(1))

	// Run the graph
	g.Run()

	// Send inputs into the graph
	go func(g *services.Graph) {
		for i := uint32(0); i < g.GetBatchSize(); i++ {
			g.Send(services.NewChunk(i, i+1))
		}
	}(g)

	// Get the output
	s := g.GetStream().(*RevealStream)

	ok := true
	var chunk services.Chunk

	for ok {
		chunk, ok = g.GetOutput()
		for i := chunk.Begin(); i < chunk.End(); i++ {
			// Compute expected result for this slot
			cryptops.RootCoprime(s.Grp, CypherMsgExpected.Get(i), s.Z, CypherMsgExpected.Get(i))

			// Execute root coprime on the keys for the Associated Data
			cryptops.RootCoprime(s.Grp, CypherADExpected.Get(i), s.Z, CypherADExpected.Get(i))

			if CypherMsgExpected.Get(i).Cmp(s.CypherMsg.Get(i)) != 0 {
				t.Error(fmt.Sprintf("PrecompReveal: Message Keys Cypher not equal on slot %v expected %v received %v",
					i, CypherMsgExpected.Get(i).Text(16), s.CypherMsg.Get(i).Text(16)))
			}

			if CypherADExpected.Get(i).Cmp(s.CypherAD.Get(i)) != 0 {
				t.Error(fmt.Sprintf("PrecompReveal: AD Keys Cypher not equal on slot %v expected %v received %v",
					i, CypherADExpected.Get(i).Text(16), s.CypherAD.Get(i).Text(16)))
			}
		}
	}
}

func initRevealGroup() *cyclic.Group {
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
	return grp
}

func initRevealRoundBuffer(grp *cyclic.Group, batchSize uint32) *round.Buffer {
	return round.NewBuffer(grp, batchSize, batchSize)
}
