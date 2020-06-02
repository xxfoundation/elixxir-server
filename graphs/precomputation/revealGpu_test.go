////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

//+build linux,gpu

package precomputation

import (
	"fmt"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/gpumaths"
	"gitlab.com/elixxir/server/graphs"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/services"
	"sync/atomic"
	"testing"
	"time"
)

func TestRevealGpuGraph(t *testing.T) {
	grp := initRevealGroup()

	batchSize := uint32(100)

	expectedName := "PrecompRevealGPU"

	// Show that the Inti function meets the function type
	var graphInit graphs.Initializer
	graphInit = InitRevealGPUGraph

	PanicHandler := func(g, m string, err error) {
		panic(fmt.Sprintf("Error in module %s of graph %s: %s", g, m, err.Error()))
	}

	gc := services.NewGraphGenerator(4, 2, 1, 1.0)

	// Initialize graph
	g := graphInit(gc)

	if g.GetName() != expectedName {
		t.Errorf("PrecompRevealGPU has incorrect name Expected %s, Recieved %s", expectedName, g.GetName())
	}

	// Build the graph
	g.Build(batchSize, PanicHandler)

	var done *uint32
	done = new(uint32)
	*done = 0

	// Build the roundBuffer
	roundBuffer := round.NewBuffer(grp, g.GetBatchSize(), g.GetExpandedBatchSize())

	//Link the graph to the round. building the stream object
	streamPool, err := gpumaths.NewStreamPool(2, 65536)
	if err != nil {
		t.Fatal(err)
	}

	g.Link(grp, roundBuffer, nil, nil, streamPool)

	stream := g.GetStream().(*RevealStream)

	//fill the fields of the stream object for testing
	grp.FindSmallCoprimeInverse(stream.Z, 256)

	for i := uint32(0); i < g.GetExpandedBatchSize(); i++ {
		grp.RandomCoprime(stream.CypherPayloadA.Get(i))
		grp.RandomCoprime(stream.CypherPayloadB.Get(i))
	}

	// Build i/o used for testing
	CypherPayloadAExpected := grp.NewIntBuffer(g.GetExpandedBatchSize(), grp.NewInt(1))
	CypherPayloadBExpected := grp.NewIntBuffer(g.GetExpandedBatchSize(), grp.NewInt(1))

	for i := uint32(0); i < batchSize; i++ {
		s := stream

		// Compute expected result for this slot
		cryptops.RootCoprime(grp, s.CypherPayloadA.Get(i), s.Z, CypherPayloadAExpected.Get(i))
		cryptops.RootCoprime(grp, s.CypherPayloadB.Get(i), s.Z, CypherPayloadBExpected.Get(i))
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

			// Only the case for the permute graph! Right?
			if d == 0 {
				t.Error("Reveal: should not be outputting until all inputs are inputted")
			}

			if stream.CypherPayloadA.Get(i).Cmp(CypherPayloadAExpected.Get(i)) != 0 {
				t.Error(fmt.Sprintf("Reveal: CypherPayloadA slot %v not computed correctly", i))
			}
			if stream.CypherPayloadB.Get(i).Cmp(CypherPayloadBExpected.Get(i)) != 0 {
				t.Error(fmt.Sprintf("Reveal: CypherPayloadB slot %v not computed correctly", i))
			}

		}
	}

	// Wait for panics from cgbn to occur, in case the modular inverse doesn't exist
	time.Sleep(time.Second)
}
