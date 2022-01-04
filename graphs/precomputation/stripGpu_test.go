///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

//+build linux,gpu

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

// Shows that results from the strip GPU kernel are the same as those that
// should be expected from running Strip phase manually
func TestStripGPU_Graph(t *testing.T) {
	viper.Set("useGPU", true)
	grp := initStripGroup()

	batchSize := uint32(100)

	// Show that the Init function meets the function type
	var graphInit graphs.Initializer
	graphInit = InitStripGPUGraph

	PanicHandler := func(g, m string, err error) {
		panic(fmt.Sprintf("Error in module %s of graph %s: %s", g, m, err.Error()))
	}

	gc := services.NewGraphGenerator(4, uint8(runtime.NumCPU()), services.AutoOutputSize, 0)

	// Initialize graph
	g := graphInit(gc)

	expectedName := "PrecompStripGPU"

	if g.GetName() != expectedName {
		t.Errorf("PrecompStripGPU has incorrect name Expected %s, Received %s", expectedName, g.GetName())
	}

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
	streamPool, err := gpumaths.NewStreamPool(2, 65536)
	if err != nil {
		t.Fatal(err)
	}

	g.Link(grp, roundBuffer, nil, streamPool)

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
				t.Error(fmt.Sprintf("PrecompStrip: PayloadA Keys Precomp not equal on slot %v expected %v received %v",
					i, PayloadAPrecomputationExpected.Get(i).Text(16), s.PayloadAPrecomputation.Get(i).Text(16)))
			}

			if PayloadBPrecomputationExpected.Get(i).Cmp(s.PayloadBPrecomputation.Get(i)) != 0 {
				t.Error(fmt.Sprintf("PrecompStrip: PayloadB Keys Precomp not equal on slot %v expected %v received %v",
					i, PayloadBPrecomputationExpected.Get(i).Text(16), s.PayloadBPrecomputation.Get(i).Text(16)))
			}
		}
	}
}
