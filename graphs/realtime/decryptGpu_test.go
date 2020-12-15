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
	"gitlab.com/elixxir/crypto/cmix"
	hash2 "gitlab.com/elixxir/crypto/hash"
	gpumaths "gitlab.com/elixxir/gpumathsgo"
	"gitlab.com/elixxir/server/cryptops"
	"gitlab.com/elixxir/server/graphs"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/services"
	"gitlab.com/xx_network/primitives/id"
	"golang.org/x/crypto/blake2b"
	"runtime"
	"testing"
)

// This test is largely similar to TestDecryptStreamInGraph,
// except it uses the GPU graph instead.
func TestDecryptStreamInGraphGPU(t *testing.T) {

	instance := mockServerInstance(t)
	grp := instance.GetConsensus().GetCmixGroup()
	registry := instance.GetUserRegistry()
	u := registry.NewUser(grp)
	u.IsRegistered = true
	registry.UpsertUser(u)

	// Reception base key should be around 256 bits long,
	// depending on generation, to feed the 256-bit hash
	if u.BaseKey.BitLen() < 250 || u.BaseKey.BitLen() > 256 {
		t.Errorf("Base key has wrong number of bits. "+
			"Had %v bits in reception base key",
			u.BaseKey.BitLen())
	}

	//var stream DecryptStream
	batchSize := uint32(32)
	//stream.Link(batchSize, &node.RoundBuffer{Grp: grp})

	// make a salt for testing
	testSalt := []byte("sodium chloride")
	// pad to length of the base key
	testSalt = append(testSalt, make([]byte, 256/8-len(testSalt))...)

	PanicHandler := func(g, m string, err error) {
		panic(fmt.Sprintf("Error in module %s of graph %s: %s", g, m, err.Error()))
	}

	// Show that the Init function meets the function type
	var graphInit graphs.Initializer
	graphInit = InitDecryptGPUGraph

	gc := services.NewGraphGenerator(4, uint8(runtime.NumCPU()), 1, 1.0)

	//Initialize graph
	g := graphInit(gc)

	g.Build(batchSize, PanicHandler)

	// Build the roundBuffer
	roundBuffer := round.NewBuffer(grp, g.GetBatchSize(), g.GetExpandedBatchSize())

	// Fill the fields of the roundBuffer object for testing
	for i := uint32(0); i < g.GetExpandedBatchSize(); i++ {

		grp.Set(roundBuffer.R.Get(i), grp.NewInt(int64(2*i+1)))
		grp.Set(roundBuffer.S.Get(i), grp.NewInt(int64(3*i+1)))
		grp.Set(roundBuffer.PayloadBPrecomputation.Get(i), grp.NewInt(int64(1)))
		grp.Set(roundBuffer.PayloadAPrecomputation.Get(i), grp.NewInt(int64(1)))

	}

	//Link the graph to the round. building the stream object
	streamPool, err := gpumaths.NewStreamPool(2, 65536)
	if err != nil {
		t.Fatal(err)
	}

	g.Link(grp, roundBuffer, registry, round.NewClientFailureReport(), streamPool)

	stream := g.GetStream().(*KeygenDecryptStream)

	expectedPayloadA := grp.NewIntBuffer(g.GetExpandedBatchSize(), grp.NewInt(1))
	expectedPayloadB := grp.NewIntBuffer(g.GetExpandedBatchSize(), grp.NewInt(1))

	kmacHash, err := hash2.NewCMixHash()
	if err != nil {
		t.Errorf("Could not get hash for KMACing")
	}

	// So, it's necessary to fill in the parts in the expanded batch with dummy
	// data to avoid crashing, or we need to exclude those parts in the cryptop
	for i := 0; i < int(g.GetExpandedBatchSize()); i++ {
		// Necessary to avoid crashing
		stream.Users[i] = &id.ZeroUser
		// Not necessary to avoid crashing
		stream.Salts[i] = []byte{}

		grp.SetUint64(stream.EcrPayloadA.Get(uint32(i)), uint64(i+1))
		grp.SetUint64(stream.EcrPayloadB.Get(uint32(i)), uint64(1000+i))

		grp.SetUint64(expectedPayloadA.Get(uint32(i)), uint64(i+1))
		grp.SetUint64(expectedPayloadB.Get(uint32(i)), uint64(1000+i))

		stream.Salts[i] = testSalt
		stream.Users[i] = u.ID
		stream.KMACS[i] = [][]byte{cmix.GenerateKMAC(testSalt, u.BaseKey, kmacHash)}
	}
	// Here's the actual data for the test

	g.Run()
	go g.Send(services.NewChunk(0, g.GetExpandedBatchSize()), nil)

	ok := true
	var chunk services.Chunk
	hash, _ := blake2b.New256(nil)

	for ok {
		chunk, ok = g.GetOutput()

		for i := chunk.Begin(); i < chunk.End(); i++ {
			keyA := grp.NewInt(1)
			keyB := grp.NewInt(1)

			user, _ := registry.GetUser(stream.Users[i], grp)

			cryptops.Keygen(grp, stream.Salts[i], user.BaseKey, keyA)

			hash.Reset()
			hash.Write(stream.Salts[i])

			cryptops.Keygen(grp, hash.Sum(nil), user.BaseKey, keyB)

			// Verify expected KeyA matches actual KeyPayloadA
			if stream.KeysPayloadA.Get(i).Cmp(keyA) != 0 {
				t.Error(fmt.Sprintf("RealtimeDecrypt: Payload A Keys not equal on slot %v expected %v received %v",
					i, keyA.Text(16), stream.KeysPayloadA.Get(i).Text(16)))
			}

			// Verify expected KeyB matches actual KeyPayloadB
			if stream.KeysPayloadB.Get(i).Cmp(keyB) != 0 {
				t.Error(fmt.Sprintf("RealtimeDecrypt: Payload B Keys not equal on slot %v expected %v received %v",
					i, keyB.Text(16), stream.KeysPayloadB.Get(i).Text(16)))
			}

			cryptops.Mul3(grp, keyA, stream.R.Get(i), expectedPayloadA.Get(i))
			cryptops.Mul3(grp, keyB, stream.U.Get(i), expectedPayloadB.Get(i))

			// test that expectedPayloadA.Get(i) == stream.EcrPayloadA.Get(i)
			if stream.EcrPayloadA.Get(i).Cmp(expectedPayloadA.Get(i)) != 0 {
				t.Error(fmt.Sprintf("RealtimeDecrypt: Ecr PayloadA not equal on slot %v expected %v received %v",
					i, expectedPayloadA.Get(i).Text(16), stream.EcrPayloadA.Get(i).Text(16)))
			}

			// test that expectedPayloadB.Get(i) == stream.EcrPayloadB.Get(i)
			if stream.EcrPayloadB.Get(i).Cmp(expectedPayloadB.Get(i)) != 0 {
				t.Error(fmt.Sprintf("RealtimeDecrypt: Ecr PayloadB not equal on slot %v expected %v received %v",
					i, expectedPayloadB.Get(i).Text(16), stream.EcrPayloadB.Get(i).Text(16)))
			}
		}
	}
}
