package precomputation

import (
	"fmt"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/server/graphs"
	"gitlab.com/elixxir/server/node"
	"gitlab.com/elixxir/server/services"
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

	rs := RevealStream{}

	batchSize := uint32(100)

	round := node.NewRound(grp, 1, batchSize, batchSize)

	rs.Link(batchSize, round)

	if round.Z.Cmp(rs.Z) != 0 {
		t.Errorf(
			"RevealStream.Link() Z value not linked: Expected %s, Recieved %s",
			round.Z.TextVerbose(10, 16), rs.Z.TextVerbose(10,16))
	}

	checkIntBuffer(rs.CypherMsg, batchSize, "CypherMsg", grp.NewInt(1), t)
	checkIntBuffer(rs.CypherAD, batchSize, "CypherAD", grp.NewInt(1), t)
}

func TestReveal_Graph(t *testing.T) {
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
	graphInit = InitRevealGraph

	// Initialize graph
	g := graphInit(func(err error) { return })

	// Build the graph
	g.Build(batchSize, services.AUTO_OUTPUTSIZE, 0)

	// Build the round
	round := node.NewRound(grp, 1, g.GetBatchSize(), g.GetExpandedBatchSize())

	// Link the graph to the round. building the stream object
	g.Link(round)

	stream := g.GetStream().(*RevealStream)

	// Fill the fields of the stream object for testing
	grp.Random(stream.CypherPublicKey)

	grp.Random(stream.Z)

	// Build i/o used for testing
	CypherMsgExpected := grp.NewIntBuffer(g.GetExpandedBatchSize(), grp.NewInt(1))
	CypherADExpected := grp.NewIntBuffer(g.GetExpandedBatchSize(), grp.NewInt(1))

	// Run the graph
	g.Run()

	// Send inputs into the graph
	go func(g *services.Graph) {
		for i := uint32(0); i < g.GetExpandedBatchSize(); i++ {
			g.Send(services.NewChunk(i, i+1))
		}
	}(g)

	// Get the output
	s := g.GetStream().(*RevealStream)

	for chunk := range g.ChunkDoneChannel() {
		for i := chunk.Begin(); i < chunk.End(); i++ {
			// Compute expected result for this slot
			cryptops.RootCoprime(s.Grp, CypherMsgExpected.Get(i), s.Z, CypherMsgExpected.Get(i))

			// Execute root coprime on the keys for the Associated Data
			cryptops.RootCoprime(s.Grp, CypherADExpected.Get(i), s.Z, CypherADExpected.Get(i))

			if CypherMsgExpected.Get(i).Cmp(s.CypherMsg.Get(i)) != 0 {
				t.Error(fmt.Sprintf("PrecompReveal: Message Keys Cypher not equal on slot %v", i))
			}

			if CypherADExpected.Get(i).Cmp(s.CypherAD.Get(i)) != 0 {
				t.Error(fmt.Sprintf("PrecompReveal: AD Keys Cypher not equal on slot %v", i))
			}
		}
	}
}

