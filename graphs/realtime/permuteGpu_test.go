///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

//+build linux,cgo,gpu

package realtime

import (
	"fmt"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/shuffle"
	gpumaths "gitlab.com/elixxir/gpumathsgo"
	"gitlab.com/elixxir/server/cryptops"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/xx_network/crypto/large"
	"runtime"
	"sync/atomic"
	"testing"
)

// This test is largely similar to TestPermuteStreamInGraph,
// except it uses the GPU graph instead.
func TestPermuteStream_InGraphGPU(t *testing.T) {
	viper.Set("useGpu", true)
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
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2))

	batchSize := uint32(10)

	PanicHandler := func(g, m string, err error) {
		panic(fmt.Sprintf("Error in module %s of graph %s: %s", g, m, err.Error()))
	}

	gc := services.NewGraphGenerator(4, uint8(runtime.NumCPU()), 1, 1.0)

	g := InitPermuteGPUGraph(gc)

	g.Build(batchSize, PanicHandler)

	var done *uint32
	done = new(uint32)
	*done = 0

	roundBuffer := round.NewBuffer(grp, batchSize, g.GetExpandedBatchSize())

	subPermutation := roundBuffer.Permutations[:batchSize]

	shuffle.Shuffle32(&subPermutation)

	//Link the graph to the round. building the stream object
	streamPool, err := gpumaths.NewStreamPool(2, 65536)
	if err != nil {
		t.Fatal(err)
	}

	g.Link(grp, roundBuffer, nil, nil, streamPool)

	permuteInverse := make([]uint32, g.GetExpandedBatchSize())
	for i := uint32(0); i < uint32(len(permuteInverse)); i++ {
		permuteInverse[roundBuffer.Permutations[i]] = i
	}

	ps := g.GetStream().(*PermuteStream)

	expectedPayloadA := grp.NewIntBuffer(batchSize, grp.NewInt(1))
	expectedPayloadB := grp.NewIntBuffer(batchSize, grp.NewInt(1))

	for i := uint32(0); i < batchSize; i++ {
		grp.SetUint64(ps.EcrPayloadA.Get(i), uint64(i+1))
		grp.SetUint64(ps.EcrPayloadB.Get(i), uint64(i+1001))
	}

	for i := uint32(0); i < batchSize; i++ {
		grp.Set(expectedPayloadA.Get(i), ps.EcrPayloadA.Get(i))
		grp.Set(expectedPayloadB.Get(i), ps.EcrPayloadB.Get(i))

		cryptops.Mul2(grp, ps.S.Get(i), expectedPayloadA.Get(i))
		cryptops.Mul2(grp, ps.V.Get(i), expectedPayloadB.Get(i))

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

			// Compute expected result for this slot
			if ps.PayloadAPermuted[i].Cmp(expectedPayloadA.Get(permuteInverse[i])) != 0 {
				t.Error(fmt.Sprintf("Permute: Slot %v out1 not permuted correctly", i))
			}

			if ps.PayloadBPermuted[i].Cmp(expectedPayloadB.Get(permuteInverse[i])) != 0 {
				t.Error(fmt.Sprintf("Permute: Slot %v out2 not permuted correctly", i))
			}
		}
	}
}
