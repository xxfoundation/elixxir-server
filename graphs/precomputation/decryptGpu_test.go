////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

//+build linux,cgo,gpu

package precomputation

import (
	"fmt"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/gpumathsgo"
	"gitlab.com/elixxir/gpumathsgo/cryptops"
	"gitlab.com/elixxir/server/graphs"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/services"
	"runtime"
	"testing"
)

// Runs precomp decrypt test with GPU stream pool and graphs
func TestDecryptGPUGraph(t *testing.T) {
	viper.Set("useGPU", true)
	grp := initDecryptGroup()

	batchSize := uint32(100)

	expectedName := "PrecompDecryptGPU"

	//Show that the Inti function meets the function type
	var graphInit graphs.Initializer
	graphInit = InitDecryptGPUGraph

	PanicHandler := func(g, m string, err error) {
		panic(fmt.Sprintf("Error in module %s of graph %s: %s", g, m, err.Error()))
	}

	gc := services.NewGraphGenerator(4, uint8(runtime.NumCPU()), services.AutoOutputSize, 1.0)

	//Initialize graph
	g := graphInit(gc)

	if g.GetName() != expectedName {
		t.Errorf("PrecompDecrypt has incorrect name Expected %s, Received %s", expectedName, g.GetName())
	}

	//Build the graph
	g.Build(batchSize, PanicHandler)

	//Build the round
	roundBuffer := round.NewBuffer(grp, g.GetBatchSize(), g.GetExpandedBatchSize())

	//Link the graph to the round. building the stream object
	streamPool, err := gpumaths.NewStreamPool(2, 65536)
	if err != nil {
		t.Fatal(err)
	}

	g.Link(grp, roundBuffer, nil, streamPool)

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
