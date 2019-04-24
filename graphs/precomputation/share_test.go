package precomputation

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/server/graphs"
	"gitlab.com/elixxir/server/node"
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
		t.Errorf("ShareStream.GetName(), Expected %s, Recieved %s", expected, rs.GetName())
	}
}

// Test that RevealStream.Link() Links correctly
func TestShareStream_Link(t *testing.T) {
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

	ss := ShareStream{}

	batchSize := uint32(100)

	round := node.NewRound(grp, batchSize, batchSize)

	ss.Link(batchSize, round)

	if round.Z.Cmp(ss.Z) != 0 {
		t.Errorf(
			"ShareStream.Link() Z value not linked: Expected %s, Recieved %s",
			round.Z.TextVerbose(10, 16), ss.Z.TextVerbose(10, 16))
	}

	// Edit round to show that Z value in stream changes
	expected := grp.Random(round.Z)

	if ss.Z.Cmp(expected) != 0 {
		t.Errorf(
			"RevealStream.Link() Z value not linked to round: Expected %s, Recieved %s",
			round.Z.TextVerbose(10, 16), ss.Z.TextVerbose(10, 16))
	}

	if ss.PartialPublicCypherKey == nil {
		t.Errorf(
			"ShareStream.Link(): Partial Cypher Key not set")
	}
}

// Tests Input's happy path
func TestShareStream_Input(t *testing.T) {
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

	ss := &ShareStream{}

	round := node.NewRound(grp, batchSize, batchSize)

	ss.Link(batchSize, round)

	msg := &mixmessages.CmixSlot{
		PartialRoundPublicCypherKey: []byte{1},
	}

	err := ss.Input(0, msg)

	if err != nil {
		t.Errorf("RevealStream.Input() errored: %s", err.Error())
	}

	if ss.PartialPublicCypherKey.Cmp(grp.NewInt(1)) != 0 {
		t.Errorf("ShareStream.Input() partialPublicCypherKey not set to 1: %s",
			ss.PartialPublicCypherKey.Text(16))
	}
}

// Tests that Input errors correct when the passed value is out of the group
func TestShareStream_Input_OutOfGroup(t *testing.T) {
	grp := cyclic.NewGroup(large.NewInt(11), large.NewInt(4), large.NewInt(5))

	batchSize := uint32(100)

	ss := &ShareStream{}

	round := node.NewRound(grp, batchSize, batchSize)

	ss.Link(batchSize, round)

	msg := &mixmessages.CmixSlot{
		PartialRoundPublicCypherKey: []byte{0},
	}

	err := ss.Input(0, msg)

	if err != node.ErrOutsideOfGroup {
		t.Errorf("SharetStream.Input() did not return an error when out of group")
	}
}

// Tests that the output function returns a valid cmixMessage
func TestShareStream_Output(t *testing.T) {
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

	ss := &ShareStream{}

	round := node.NewRound(grp, batchSize, batchSize)

	ss.Link(batchSize, round)

	expected := []byte{1}

	msg := &mixmessages.CmixSlot{
		PartialRoundPublicCypherKey: expected,
	}

	err := ss.Input(0, msg)

	if err != nil {
		t.Errorf("RevealStream.Output() errored on input: %s", err.Error())
	}

	output := ss.Output(0)

	if !reflect.DeepEqual(output.PartialRoundPublicCypherKey, expected) {
		t.Errorf("RevealStream.Output() incorrect recieved Partial Round Cypher Key: Expected: %v, Recieved: %v",
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

	batchSize := uint32(1)

	// Show that the Init function meets the function type
	var graphInit graphs.Initializer
	graphInit = InitShareGraph

	PanicHandler := func(err error) {
		t.Errorf("Share: Error in adapter: %s", err.Error())
		return
	}

	gc := services.NewGraphGenerator(1, PanicHandler, uint8(runtime.NumCPU()))

	//Initialize graph
	g := graphInit(gc)

	// Build the graph
	g.Build(batchSize, services.AUTO_OUTPUTSIZE, 0)

	// Build the round
	round := node.NewRound(grp, g.GetBatchSize(), g.GetExpandedBatchSize())

	// Fill the fields of the stream object for testing
	grp.FindSmallCoprimeInverse(round.Z, 256)

	// Link the graph to the round. building the stream object
	g.Link(round)

	stream := g.GetStream().(*ShareStream)
	grp.SetUint64(stream.PartialPublicCypherKey, 2)

	// Build i/o used for testing
	PubicCypherKeyExpected := grp.ExpG(round.Z, grp.NewInt(1))

	// Run the graph
	g.Run()

	// Send inputs into the graph
	go func(g *services.Graph) {

		g.Send(services.NewChunk(0, 1))
	}(g)

	ok := true
	var chunk services.Chunk

	for ok {
		chunk, ok = g.GetOutput()
		for i := chunk.Begin(); i < chunk.End(); i++ {
			if PubicCypherKeyExpected.Cmp(stream.PartialPublicCypherKey) != 0 {
				t.Errorf("PrecompShare:PartialPublicCypherKey incorrect, Expected: %v, Recieved: %v",
					PubicCypherKeyExpected.Text(16), stream.PartialPublicCypherKey.Text(16))
			}
		}
	}
}
