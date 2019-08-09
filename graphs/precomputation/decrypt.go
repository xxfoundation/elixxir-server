////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package precomputation

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/services"
)

// This file implements the Graph for the Precomputation Decrypt phase
// Decrypt phase transforms first unpermuted internode keys
// and partial cypher texts into the data that the permute phase needs

// DecryptStream holds data containing keys and inputs used by decrypt
type DecryptStream struct {
	Grp             *cyclic.Group
	PublicCypherKey *cyclic.Int

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

	ds.LinkPrecompDecryptStream(grp, batchSize, roundBuffer,
		grp.NewIntBuffer(batchSize, grp.NewInt(1)),
		grp.NewIntBuffer(batchSize, grp.NewInt(1)),
		grp.NewIntBuffer(batchSize, grp.NewInt(1)),
		grp.NewIntBuffer(batchSize, grp.NewInt(1)),
	)
}

func (ds *DecryptStream) LinkPrecompDecryptStream(grp *cyclic.Group, batchSize uint32, roundBuffer *round.Buffer,
	keysPayloadA, cypherPayloadA, keysPayloadB, cypherPayloadB *cyclic.IntBuffer) {

	ds.Grp = grp
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

			// Execute elgamal on the keys for the Message
			elgamal(ds.Grp, ds.R.Get(i), ds.Y_R.Get(i), ds.PublicCypherKey, ds.KeysPayloadA.Get(i), ds.CypherPayloadA.Get(i))

			// Execute elgamal on the keys for the Associated Data
			elgamal(ds.Grp, ds.U.Get(i), ds.Y_U.Get(i), ds.PublicCypherKey, ds.KeysPayloadB.Get(i), ds.CypherPayloadB.Get(i))
		}
		return nil
	},
	Cryptop:    cryptops.ElGamal,
	NumThreads: services.AutoNumThreads,
	InputSize:  services.AutoInputSize,
	Name:       "DecryptElgamal",
}

// InitDecryptGraph is called to initialize the graph. Conforms to graphs.Initialize function type
func InitDecryptGraph(gc services.GraphGenerator) *services.Graph {
	g := gc.NewGraph("PrecompDecrypt", &DecryptStream{})

	decryptElgamal := DecryptElgamal.DeepCopy()

	g.First(decryptElgamal)
	g.Last(decryptElgamal)

	return g
}
