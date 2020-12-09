///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

//+build linux,cgo,gpu

package precomputation

import (
	"fmt"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/shuffle"
	"gitlab.com/elixxir/gpumathsgo"
	"gitlab.com/elixxir/server/graphs"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/services"
	"sync/atomic"
	"testing"
)

func TestPermuteGpuGraph(t *testing.T) {
	viper.Set("useGpu", true)
	grp := initPermuteGroup()

	batchSize := uint32(100)

	expectedName := "PrecompPermuteGPU"

	// Show that the Inti function meets the function type
	var graphInit graphs.Initializer
	graphInit = InitPermuteGPUGraph

	PanicHandler := func(g, m string, err error) {
		panic(fmt.Sprintf("Error in module %s of graph %s: %s", g, m, err.Error()))
	}

	gc := services.NewGraphGenerator(4, 2, 1, 1.0)

	// Initialize graph
	g := graphInit(gc)

	if g.GetName() != expectedName {
		t.Errorf("PrecompPermuteGPU has incorrect name Expected %s, Received %s", expectedName, g.GetName())
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

	//Link the graph to the round. building the stream object
	streamPool, err := gpumaths.NewStreamPool(2, 65536)
	if err != nil {
		t.Fatal(err)
	}

	g.Link(grp, roundBuffer, nil, nil, streamPool)

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
