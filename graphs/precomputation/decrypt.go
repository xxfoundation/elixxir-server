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
	KeysMsg   *cyclic.IntBuffer
	CypherMsg *cyclic.IntBuffer
	KeysAD    *cyclic.IntBuffer
	CypherAD  *cyclic.IntBuffer
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
	keysMsg, cypherMsg, keysAD, cypherAD *cyclic.IntBuffer) {

	ds.Grp = grp
	ds.PublicCypherKey = roundBuffer.CypherPublicKey

	ds.R = roundBuffer.R.GetSubBuffer(0, batchSize)
	ds.U = roundBuffer.U.GetSubBuffer(0, batchSize)
	ds.Y_R = roundBuffer.Y_R.GetSubBuffer(0, batchSize)
	ds.Y_U = roundBuffer.Y_U.GetSubBuffer(0, batchSize)

	ds.KeysMsg = keysMsg
	ds.CypherMsg = cypherMsg
	ds.KeysAD = keysAD
	ds.CypherAD = cypherAD

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

	if index >= uint32(ds.KeysMsg.Len()) {
		return services.ErrOutsideOfBatch
	}

	if !ds.Grp.BytesInside(slot.EncryptedMessageKeys, slot.PartialMessageCypherText,
		slot.EncryptedAssociatedDataKeys, slot.PartialAssociatedDataCypherText) {
		return services.ErrOutsideOfGroup
	}

	ds.Grp.SetBytes(ds.KeysMsg.Get(index), slot.EncryptedMessageKeys)
	ds.Grp.SetBytes(ds.KeysAD.Get(index), slot.EncryptedAssociatedDataKeys)
	ds.Grp.SetBytes(ds.CypherMsg.Get(index), slot.PartialMessageCypherText)
	ds.Grp.SetBytes(ds.CypherAD.Get(index), slot.PartialAssociatedDataCypherText)

	return nil
}

// Output returns a cmix slot message
func (ds *DecryptStream) Output(index uint32) *mixmessages.Slot {

	return &mixmessages.Slot{
		Index:                           index,
		EncryptedMessageKeys:            ds.KeysMsg.Get(index).Bytes(),
		EncryptedAssociatedDataKeys:     ds.KeysAD.Get(index).Bytes(),
		PartialMessageCypherText:        ds.CypherMsg.Get(index).Bytes(),
		PartialAssociatedDataCypherText: ds.CypherAD.Get(index).Bytes(),
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
			elgamal(ds.Grp, ds.R.Get(i), ds.Y_R.Get(i), ds.PublicCypherKey, ds.KeysMsg.Get(i), ds.CypherMsg.Get(i))

			// Execute elgamal on the keys for the Associated Data
			elgamal(ds.Grp, ds.U.Get(i), ds.Y_U.Get(i), ds.PublicCypherKey, ds.KeysAD.Get(i), ds.CypherAD.Get(i))
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
