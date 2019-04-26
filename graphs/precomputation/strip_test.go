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
	"gitlab.com/elixxir/server/node"
	"gitlab.com/elixxir/server/services"
	"reflect"
	"runtime"
	"testing"
)

// Test that StripStream.GetName() returns the correct name
func TestStripStream_GetName(t *testing.T) {
	expected := "PrecompStripStream"

	stream := StripStream{}

	if stream.GetName() != expected {
		t.Errorf("StripStream.GetName(), Expected %s, Recieved %s", expected, stream.GetName())
	}
}

// Test that StripStream.Link() Links correctly
func TestStripStream_Link(t *testing.T) {
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

	stream := StripStream{}

	batchSize := uint32(100)

	round := node.NewRound(grp, 1, batchSize, batchSize)

	stream.Link(batchSize, round)

	if round.Z.Cmp(stream.Z) != 0 {
		t.Errorf(
			"RevealStream.Link() Z value not linked: Expected %s, Recieved %s",
			round.Z.TextVerbose(10, 16), stream.Z.TextVerbose(10, 16))
	}

	checkStreamIntBuffer(grp, stream.MessagePrecomputation, round.MessagePrecomputation, "MessagePrecomputation", t)
	checkStreamIntBuffer(grp, stream.ADPrecomputation, round.ADPrecomputation, "ADPrecomputation", t)

	checkIntBuffer(stream.CypherMsg, batchSize, "CypherMsg", grp.NewInt(1), t)
	checkIntBuffer(stream.CypherAD, batchSize, "CypherAD", grp.NewInt(1), t)

	// Edit round to show that Z value in stream changes
	expected := grp.Random(round.Z)

	if stream.Z.Cmp(expected) != 0 {
		t.Errorf(
			"StripStream.Link() Z value not linked to round: Expected %s, Recieved %s",
			round.Z.TextVerbose(10, 16), stream.Z.TextVerbose(10, 16))
	}
}

// Tests Input's happy path
func TestStripStream_Input(t *testing.T) {
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

	batchSize := uint32(100)

	ss := &StripStream{}

	round := node.NewRound(grp, 1, batchSize, batchSize)

	ss.Link(batchSize, round)

	for b := uint32(0); b < batchSize; b++ {

		expected := [][]byte{
			{byte(1)},
			{byte(1)},
		}

		msg := &mixmessages.Slot{
			PartialMessageCypherText:        expected[0],
			PartialAssociatedDataCypherText: expected[1],
		}

		err := ss.Input(b, msg)
		if err != nil {
			t.Errorf("StripStream.Input() errored on slot %v: %s", b, err.Error())
		}

		if !reflect.DeepEqual(ss.CypherMsg.Get(b).Bytes(), expected[0]) {
			t.Errorf("StripStream.Input() incorrect stored CypherMsg data at %v: Expected: %v, Recieved: %v",
				b, expected[0], ss.CypherMsg.Get(b).Bytes())
		}

		if !reflect.DeepEqual(ss.CypherAD.Get(b).Bytes(), expected[1]) {
			t.Errorf("StripStream.Input() incorrect stored CypherAD data at %v: Expected: %v, Recieved: %v",
				b, expected[1], ss.CypherAD.Get(b).Bytes())
		}

	}

}

// Tests that the input errors correctly when the index is outside of the batch
func TestStripStream_Input_OutOfBatch(t *testing.T) {
	primeString :=
		"FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
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

	batchSize := uint32(100)

	stream := &StripStream{}

	round := node.NewRound(grp, 1, batchSize, batchSize)

	stream.Link(batchSize, round)

	msg := &mixmessages.Slot{
		PartialMessageCypherText:        []byte{0},
		PartialAssociatedDataCypherText: []byte{0},
	}

	err := stream.Input(batchSize, msg)

	if err != node.ErrOutsideOfBatch {
		t.Errorf("StripStream.Input() did nto return an outside of batch error when out of batch")
	}

	err1 := stream.Input(batchSize+1, msg)

	if err1 != node.ErrOutsideOfBatch {
		t.Errorf("StripStream.Input() did not return an outside of batch error when out of batch")
	}
}

// Tests that Input errors correct when the passed value is out of the group
func TestStripStream_Input_OutOfGroup(t *testing.T) {
	grp := cyclic.NewGroup(large.NewInt(11), large.NewInt(4), large.NewInt(5))

	batchSize := uint32(100)

	stream := &StripStream{}

	round := node.NewRound(grp, 1, batchSize, batchSize)

	stream.Link(batchSize, round)

	msg := &mixmessages.Slot{
		PartialMessageCypherText:        large.NewInt(89).Bytes(),
		PartialAssociatedDataCypherText: large.NewInt(13).Bytes(),
	}

	err := stream.Input(batchSize-10, msg)

	if err != node.ErrOutsideOfGroup {
		t.Errorf("StripStream.Input() did not return an error when out of group")
	}
}

// Tests that the output function returns a valid cmixMessage
func TestStripStream_Output(t *testing.T) {
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

	batchSize := uint32(100)

	stream := &StripStream{}

	round := node.NewRound(grp, 1, batchSize, batchSize)

	stream.Link(batchSize, round)

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
			t.Errorf("StripStream.Output() errored on slot %v: %s", b, err.Error())
		}

		output := stream.Output(b)

		if !reflect.DeepEqual(output.PartialMessageCypherText, expected[0]) {
			t.Errorf("StripStream.Output() incorrect recieved CypherMsg data at %v: Expected: %v, Recieved: %v",
				b, expected[2], stream.CypherMsg.Get(b).Bytes())
		}

		if !reflect.DeepEqual(output.PartialAssociatedDataCypherText, expected[1]) {
			t.Errorf("StripStream.Output() incorrect recieved CypherAD data at %v: Expected: %v, Recieved: %v",
				b, expected[3], stream.CypherAD.Get(b).Bytes())
		}

	}

}

// Tests that StripStream conforms to the CommsStream interface
func TestStripStream_CommsInterface(t *testing.T) {

	var face interface{}
	face = &StripStream{}
	_, ok := face.(node.CommsStream)

	if !ok {
		t.Errorf("StripStream: Does not conform to the CommsStream interface")
	}

}

func TestStrip_Graph(t *testing.T) {
	primeString :=
		"FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
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

	batchSize := uint32(100)

	// Show that the Init function meets the function type
	var graphInit graphs.Initializer
	graphInit = InitStripGraph

	PanicHandler := func(err error) {
		t.Errorf("Strip: Error in adaptor: %s", err.Error())
		return
	}

	gc := services.NewGraphGenerator(4, PanicHandler, uint8(runtime.NumCPU()))

	// Initialize graph
	g := graphInit(gc)

	// Build the graph
	g.Build(batchSize, services.AUTO_OUTPUTSIZE, 0)

	// Build the round
	round := node.NewRound(grp, 1, g.GetBatchSize(), g.GetExpandedBatchSize())

	// Fill the fields of the round object for testing
	for i := uint32(0); i < g.GetBatchSize(); i++ {
		grp.Set(round.ADPrecomputation.Get(i), grp.NewInt(int64(1)))
		grp.Set(round.MessagePrecomputation.Get(i), grp.NewInt(int64(1)))
	}

	grp.FindSmallCoprimeInverse(round.Z, 256)

	// Link the graph to the round. building the stream object
	g.Link(round)

	stream := g.GetStream().(*StripStream)

	// Fill the fields of the stream object for testing
	for i := uint32(0); i < g.GetBatchSize(); i++ {
		grp.RandomCoprime(stream.CypherMsg.Get(i))
		grp.RandomCoprime(stream.CypherAD.Get(i))
	}

	// Build i/o used for testing
	CypherMsgExpected := grp.NewIntBuffer(g.GetExpandedBatchSize(), grp.NewInt(1))
	CypherADExpected := grp.NewIntBuffer(g.GetExpandedBatchSize(), grp.NewInt(1))
	MessagePrecomputationExpected := grp.NewIntBuffer(g.GetExpandedBatchSize(), grp.NewInt(1))
	ADPrecomputationExpected := grp.NewIntBuffer(g.GetExpandedBatchSize(), grp.NewInt(1))

	// Run the graph
	g.Run()

	// Send inputs into the graph
	go func(g *services.Graph) {
		for i := uint32(0); i < g.GetBatchSize(); i++ {
			g.Send(services.NewChunk(i, i+1))
		}
	}(g)

	// Get the output
	s := g.GetStream().(*StripStream)

	ok := true
	var chunk services.Chunk

	for ok {
		chunk, ok = g.GetOutput()
		for i := chunk.Begin(); i < chunk.End(); i++ {
			// Compute root coprime for msg & associated data
			cryptops.RootCoprime(s.Grp, CypherMsgExpected.Get(i), s.Z, CypherMsgExpected.Get(i))
			cryptops.RootCoprime(s.Grp, CypherADExpected.Get(i), s.Z, CypherADExpected.Get(i))

			// Compute inverse
			cryptops.Inverse(s.Grp, MessagePrecomputationExpected.Get(i), MessagePrecomputationExpected.Get(i))
			cryptops.Inverse(s.Grp, ADPrecomputationExpected.Get(i), ADPrecomputationExpected.Get(i))

			// Compute mul2
			cryptops.Mul2(s.Grp, s.CypherMsg.Get(i), MessagePrecomputationExpected.Get(i))
			cryptops.Mul2(s.Grp, s.CypherAD.Get(i), ADPrecomputationExpected.Get(i))

			// Verify message and associated data match the expected values
			if CypherMsgExpected.Get(i).Cmp(s.CypherMsg.Get(i)) != 0 {
				t.Error(fmt.Sprintf("PrecompStrip: Message Keys Cypher not equal on slot %v expected %v received %v",
					i, CypherMsgExpected.Get(i).Text(16), s.CypherMsg.Get(i).Text(16)))
			}

			if CypherADExpected.Get(i).Cmp(s.CypherAD.Get(i)) != 0 {
				t.Error(fmt.Sprintf("PrecompStrip: AD Keys Cypher not equal on slot %v expected %v received %v",
					i, CypherADExpected.Get(i).Text(16), s.CypherAD.Get(i).Text(16)))
			}

			if MessagePrecomputationExpected.Get(i).Cmp(s.MessagePrecomputation.Get(i)) != 0 {
				t.Error(fmt.Sprintf("PrecompStrip: Message Keys Cypher not equal on slot %v expected %v received %v",
					i, MessagePrecomputationExpected.Get(i).Text(16), s.CypherMsg.Get(i).Text(16)))
			}

			if ADPrecomputationExpected.Get(i).Cmp(s.ADPrecomputation.Get(i)) != 0 {
				t.Error(fmt.Sprintf("PrecompStrip: AD Keys Cypher not equal on slot %v expected %v received %v",
					i, ADPrecomputationExpected.Get(i).Text(16), s.CypherAD.Get(i).Text(16)))
			}
		}
	}
}
