///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package precomputation

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/gpumathsgo"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/services"
)

// This file implements the Graph for the Precomputation Decrypt phase
// Decrypt phase transforms first unpermuted internode keys
// and partial cypher texts into the data that the permute phase needs

// DecryptStream holds data containing keys and inputs used by decrypt
type DecryptStream struct {
	Grp             *cyclic.Group
	PublicCypherKey *cyclic.Int
	StreamPool      *gpumaths.StreamPool

	// Link to round object
	R *cyclic.IntBuffer
	U *cyclic.IntBuffer

	Y_R *cyclic.IntBuffer
	Y_U *cyclic.IntBuffer

	// Unique to stream
	KeysPayloadA   *cyclic.IntBuffer
	CypherPayloadA *cyclic.IntBuffer
	KeysPayloadB   *cyclic.IntBuffer
	CypherPayloadB *cyclic.IntBuffer
}

// GetName returns stream name
func (ds *DecryptStream) GetName() string {
	return "PrecompDecryptStream"
}

// Link binds stream to state objects in round
func (ds *DecryptStream) Link(grp *cyclic.Group, batchSize uint32, source ...interface{}) {
	roundBuffer := source[0].(*round.Buffer)
	var streamPool *gpumaths.StreamPool
	if len(source) >= 4 {
		// All arguments are being passed from the Link call, which should include the stream pool
		streamPool = source[3].(*gpumaths.StreamPool)
	}

	ds.LinkPrecompDecryptStream(grp, batchSize, roundBuffer, streamPool,
		grp.NewIntBuffer(batchSize, grp.NewInt(1)),
		grp.NewIntBuffer(batchSize, grp.NewInt(1)),
		grp.NewIntBuffer(batchSize, grp.NewInt(1)),
		grp.NewIntBuffer(batchSize, grp.NewInt(1)),
	)
}

func (ds *DecryptStream) LinkPrecompDecryptStream(grp *cyclic.Group, batchSize uint32, roundBuffer *round.Buffer,
	pool *gpumaths.StreamPool, keysPayloadA, cypherPayloadA, keysPayloadB, cypherPayloadB *cyclic.IntBuffer) {

	ds.Grp = grp
	ds.StreamPool = pool
	ds.PublicCypherKey = roundBuffer.CypherPublicKey

	ds.R = roundBuffer.R.GetSubBuffer(0, batchSize)
	ds.U = roundBuffer.U.GetSubBuffer(0, batchSize)
	ds.Y_R = roundBuffer.Y_R.GetSubBuffer(0, batchSize)
	ds.Y_U = roundBuffer.Y_U.GetSubBuffer(0, batchSize)

	ds.KeysPayloadA = keysPayloadA
	ds.CypherPayloadA = cypherPayloadA
	ds.KeysPayloadB = keysPayloadB
	ds.CypherPayloadB = cypherPayloadB

}

type PrecompDecryptSubstreamInterface interface {
	GetPrecompDecryptSubStream() *DecryptStream
}

// getSubStream implements reveal interface to return stream object
func (ds *DecryptStream) GetPrecompDecryptSubStream() *DecryptStream {
	return ds
}

// Input initializes stream inputs from slot
func (ds *DecryptStream) Input(index uint32, slot *mixmessages.Slot) error {

	if index >= uint32(ds.KeysPayloadA.Len()) {
		return services.ErrOutsideOfBatch
	}

	if !ds.Grp.BytesInside(slot.EncryptedPayloadAKeys, slot.PartialPayloadACypherText,
		slot.EncryptedPayloadBKeys, slot.PartialPayloadBCypherText) {
		return services.ErrOutsideOfGroup
	}

	ds.Grp.SetBytes(ds.KeysPayloadA.Get(index), slot.EncryptedPayloadAKeys)
	ds.Grp.SetBytes(ds.KeysPayloadB.Get(index), slot.EncryptedPayloadBKeys)
	ds.Grp.SetBytes(ds.CypherPayloadA.Get(index), slot.PartialPayloadACypherText)
	ds.Grp.SetBytes(ds.CypherPayloadB.Get(index), slot.PartialPayloadBCypherText)

	return nil
}

// Output returns a cmix slot message
func (ds *DecryptStream) Output(index uint32) *mixmessages.Slot {
	return &mixmessages.Slot{
		Index:                     index,
		EncryptedPayloadAKeys:     ds.KeysPayloadA.Get(index).Bytes(),
		EncryptedPayloadBKeys:     ds.KeysPayloadB.Get(index).Bytes(),
		PartialPayloadACypherText: ds.CypherPayloadA.Get(index).Bytes(),
		PartialPayloadBCypherText: ds.CypherPayloadB.Get(index).Bytes(),
	}
}

// DecryptElgamal is the sole module in Precomputation Decrypt implementing cryptops.Elgamal
var DecryptElgamal = services.Module{
	// Multiplies in own Encrypted Keys and Partial Cypher Texts
	Adapt: func(streamInput services.Stream, cryptop cryptops.Cryptop, chunk services.Chunk) error {
		dssi, ok := streamInput.(PrecompDecryptSubstreamInterface)
		elgamal, ok2 := cryptop.(cryptops.ElGamalPrototype)

		if !ok || !ok2 {
			return errors.WithStack(services.InvalidTypeAssert)
		}

		ds := dssi.GetPrecompDecryptSubStream()

		for i := chunk.Begin(); i < chunk.End(); i++ {

			// Execute elgamal on the keys for the first payload
			elgamal(ds.Grp, ds.R.Get(i), ds.Y_R.Get(i), ds.PublicCypherKey, ds.KeysPayloadA.Get(i), ds.CypherPayloadA.Get(i))

			// Execute elgamal on the keys for the second payload
			elgamal(ds.Grp, ds.U.Get(i), ds.Y_U.Get(i), ds.PublicCypherKey, ds.KeysPayloadB.Get(i), ds.CypherPayloadB.Get(i))
		}
		return nil
	},
	Cryptop:    cryptops.ElGamal,
	NumThreads: services.AutoNumThreads,
	InputSize:  services.AutoInputSize,
	Name:       "DecryptElgamal",
}

var DecryptElgamalChunk = services.Module{
	Adapt: func(s services.Stream, c cryptops.Cryptop, chunk services.Chunk) error {
		dssi, ok := s.(PrecompDecryptSubstreamInterface)
		ec, ok2 := c.(gpumaths.ElGamalChunkPrototype)
		if !ok || !ok2 {
			return errors.WithStack(services.InvalidTypeAssert)
		}

		// Execute elgamal on the keys for the first payload
		ds := dssi.GetPrecompDecryptSubStream()
		gpuStreams := ds.StreamPool
		R := ds.R.GetSubBuffer(chunk.Begin(), chunk.End())
		yR := ds.Y_R.GetSubBuffer(chunk.Begin(), chunk.End())
		kpa := ds.KeysPayloadA.GetSubBuffer(chunk.Begin(), chunk.End())
		cpa := ds.CypherPayloadA.GetSubBuffer(chunk.Begin(), chunk.End())
		err := ec(gpuStreams, ds.Grp, R, yR, ds.PublicCypherKey, kpa, cpa)
		if err != nil {
			return err
		}

		// Execute elgamal on the keys for the second payload
		U := ds.U.GetSubBuffer(chunk.Begin(), chunk.End())
		yU := ds.Y_U.GetSubBuffer(chunk.Begin(), chunk.End())
		kpb := ds.KeysPayloadB.GetSubBuffer(chunk.Begin(), chunk.End())
		cpb := ds.CypherPayloadB.GetSubBuffer(chunk.Begin(), chunk.End())
		err = ec(gpuStreams, ds.Grp, U, yU, ds.PublicCypherKey, kpb, cpb)
		if err != nil {
			return err
		}
		return nil
	},
	Cryptop: gpumaths.ElGamalChunk,
	// Populate InputSize late, at runtime
	Name:       "DecryptElgamalChunk",
	NumThreads: 2,
}

// InitDecryptGraph is called to initialize the graph. Conforms to graphs.Initialize function type
func InitDecryptGraph(gc services.GraphGenerator) *services.Graph {
	if viper.GetBool("useGpu") {
		jww.WARN.Printf("Using precomp decrypt graph running on CPU instead of equivalent GPU graph")
	}
	g := gc.NewGraph("PrecompDecrypt", &DecryptStream{})

	decryptElgamal := DecryptElgamal.DeepCopy()

	g.First(decryptElgamal)
	g.Last(decryptElgamal)

	return g
}

func InitDecryptGPUGraph(gc services.GraphGenerator) *services.Graph {
	if !viper.GetBool("useGpu") {
		jww.WARN.Printf("Using precomp decrypt graph running on GPU instead of equivalent CPU graph")
	}
	g := gc.NewGraph("PrecompDecryptGPU", &DecryptStream{})

	decryptElgamalChunk := DecryptElgamalChunk.DeepCopy()

	g.First(decryptElgamalChunk)
	g.Last(decryptElgamalChunk)

	return g
}
