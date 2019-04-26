////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package precomputation

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/server/graphs"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/services"
	"os"
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

var prime *large.Int
var grp *cyclic.Group
var batchSize uint32

func TestMain(m *testing.M) {
	prime = large.NewIntFromString(MODP768, 16)
	grp = cyclic.NewGroup(prime, large.NewInt(5), large.NewInt(53))
	batchSize = uint32(100)
	os.Exit(m.Run())
}

//Test that GenerateStream.GetName() returns the correct name
func TestGenerateStream_GetName(t *testing.T) {
	expected := "PrecompGenerateStream"

	ds := GenerateStream{}

	if ds.GetName() != expected {
		t.Errorf("GenerateStream.GetName(), "+
			"Expected %s, Recieved %s", expected, ds.GetName())
	}
}

//Test that GenerateStream.Link() Links correctly
func TestGenerateStream_Link(t *testing.T) {
	ds := GenerateStream{}
	roundBuffer := round.NewBuffer(grp, batchSize, batchSize)
	ds.Link(grp, batchSize, roundBuffer)

	checkStreamIntBuffer(grp, ds.R, roundBuffer.R, "R", t)
	checkStreamIntBuffer(grp, ds.S, roundBuffer.S, "S", t)
	checkStreamIntBuffer(grp, ds.U, roundBuffer.U, "U", t)
	checkStreamIntBuffer(grp, ds.V, roundBuffer.V, "V", t)
	checkStreamIntBuffer(grp, ds.R, roundBuffer.R, "Y_R", t)
	checkStreamIntBuffer(grp, ds.S, roundBuffer.S, "Y_S", t)
	checkStreamIntBuffer(grp, ds.U, roundBuffer.U, "Y_U", t)
	checkStreamIntBuffer(grp, ds.V, roundBuffer.V, "Y_V", t)
}

//tests Input's happy path (Note that decrypt only sets keys and has no retvals
func TestGenerateStream_Input(t *testing.T) {
	ds := &GenerateStream{}
	roundBuffer := round.NewBuffer(grp, batchSize, batchSize)
	ds.Link(grp, batchSize, roundBuffer)

	for b := uint32(0); b < batchSize; b++ {
		msg := &mixmessages.Slot{}

		err := ds.Input(b, msg)
		if err != nil {
			t.Errorf("GenerateStream.Input() errored on slot "+
				"%v: %s", b, err.Error())
		}
	}

	checkStreamIntBuffer(grp, ds.R, roundBuffer.R, "R", t)
	checkStreamIntBuffer(grp, ds.S, roundBuffer.S, "S", t)
	checkStreamIntBuffer(grp, ds.U, roundBuffer.U, "U", t)
	checkStreamIntBuffer(grp, ds.V, roundBuffer.V, "V", t)
	checkStreamIntBuffer(grp, ds.R, roundBuffer.R, "Y_R", t)
	checkStreamIntBuffer(grp, ds.S, roundBuffer.S, "Y_S", t)
	checkStreamIntBuffer(grp, ds.U, roundBuffer.U, "Y_U", t)
	checkStreamIntBuffer(grp, ds.V, roundBuffer.V, "Y_V", t)

	msg := &mixmessages.Slot{}
	err := ds.Input(batchSize, msg)
	if err == nil {
		t.Errorf("GenerateStream.Input() didn't error on OOB slot!")
	}
}

//Tests that the output function returns a valid cmixMessage
func TestGenerateStream_Output(t *testing.T) {
	ds := &GenerateStream{}
	roundBuffer := round.NewBuffer(grp, batchSize, batchSize)
	ds.Link(grp, batchSize, roundBuffer)

	for b := uint32(0); b < batchSize; b++ {
		msg := &mixmessages.Slot{}
		err := ds.Input(b, msg)
		if err != nil {
			t.Errorf("GenerateStream.Output() errored on slot %v: %s", b, err.Error())
		}

		ds.Output(b)
	}
}

//Tests that GenerateStream conforms to the CommsStream interface
func TestGenerateStream_CommsInterface(t *testing.T) {
	var face interface{}
	face = &GenerateStream{}
	_, ok := face.(services.Stream)

	if !ok {
		t.Errorf("GenerateStream: Does not conform to the Stream interface")
	}
}

func TestGenerateGraph(t *testing.T) {
	expectedName := "PrecompGenerate"

	//Show that the init function meets the function type
	var graphInit graphs.Initializer
	graphInit = InitGenerateGraph

	PanicHandler := func(err error) {
		t.Errorf("PrecompGenerate: Error in adaptor: %s", err.Error())
		return
	}

	gc := services.NewGraphGenerator(4, PanicHandler, uint8(runtime.NumCPU()), 1, 0)

	//Initialize graph
	g := graphInit(gc)

	if g.GetName() != expectedName {
		t.Errorf("PrecompGenerate has incorrect name "+
			"Expected %s, Recieved %s", expectedName, g.GetName())
	}

	//Build the graph
	g.Build(batchSize)

	//Build the round
	roundBuffer := round.NewBuffer(grp, g.GetBatchSize(), g.GetExpandedBatchSize())

	//Link the graph to the round. building the stream object
	g.Link(grp, roundBuffer)

	//stream := g.GetStream().(*GenerateStream)

	//Run the graph
	g.Run()

	//Send inputs into the graph
	go func(g *services.Graph) {
		for i := uint32(0); i < g.GetBatchSize(); i++ {
			g.Send(services.NewChunk(i, i+1))
		}
	}(g)

	//Get the output
	s := g.GetStream().(*GenerateStream)

	ok := true

	for ok {
		_, ok = g.GetOutput()
	}

	keys := make([]*cyclic.Int, batchSize*8)
	for i := uint32(0); i < batchSize; i++ {
		kOffset := i * 8
		keys[kOffset] = s.R.Get(i)
		keys[kOffset+1] = s.S.Get(i)
		keys[kOffset+2] = s.U.Get(i)
		keys[kOffset+3] = s.V.Get(i)
		keys[kOffset+4] = s.YR.Get(i)
		keys[kOffset+5] = s.YS.Get(i)
		keys[kOffset+6] = s.YU.Get(i)
		keys[kOffset+7] = s.YV.Get(i)
	}

	for i := uint32(0); i < uint32(len(keys)); i++ {
		for j := i + 1; j < uint32(len(keys)); j++ {
			if keys[i].Cmp(keys[j]) == 0 {
				// Technically, it's possible for this to happen
				// but not often and certainly
				// not repeatedly.
				t.Errorf("Keys at index %d and %d match!",
					i, j)
			}
		}
	}
}
