////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package precomputation

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/graphs"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/services"
)

// This file implements the Graph for the Precomputation Permute phase
// Permute phase permutes the keys for both payloads with their cypher
// texts, while multiplying in its own keys.

// PermuteStream holds data containing keys and inputs used by Permute
type PermuteStream struct {
	Grp             *cyclic.Group
	PublicCypherKey *cyclic.Int

	// Link to round object
	S   *cyclic.IntBuffer // Encrypted Inverse Permuted Internode Message Key
	V   *cyclic.IntBuffer // Encrypted Inverse Permuted Internode PayloadB Key
	Y_S *cyclic.IntBuffer // Permuted Internode Message Partial Cypher Text
	Y_V *cyclic.IntBuffer // Permuted Internode PayloadB Partial Cypher Text

	// Unique to stream
	KeysPayloadA           *cyclic.IntBuffer
	KeysPayloadAPermuted   []*cyclic.Int
	CypherPayloadA         *cyclic.IntBuffer
	CypherPayloadAPermuted []*cyclic.Int
	KeysPayloadB           *cyclic.IntBuffer
	KeysPayloadBPermuted   []*cyclic.Int
	CypherPayloadB         *cyclic.IntBuffer
	CypherPayloadBPermuted []*cyclic.Int
}

// GetName returns stream name
func (ps *PermuteStream) GetName() string {
	return "PrecompPermuteStream"
}

// Link binds stream to state objects in round
func (ps *PermuteStream) Link(grp *cyclic.Group, batchSize uint32, source ...interface{}) {
	roundBuffer := source[0].(*round.Buffer)

	ps.LinkPrecompPermuteStream(grp, batchSize, roundBuffer,
		grp.NewIntBuffer(batchSize, grp.NewInt(1)),
		grp.NewIntBuffer(batchSize, grp.NewInt(1)),
		grp.NewIntBuffer(batchSize, grp.NewInt(1)),
		grp.NewIntBuffer(batchSize, grp.NewInt(1)),
		make([]*cyclic.Int, batchSize),
		make([]*cyclic.Int, batchSize),
		make([]*cyclic.Int, batchSize),
		make([]*cyclic.Int, batchSize))
}

// Link binds stream to state objects in round
func (ps *PermuteStream) LinkPrecompPermuteStream(grp *cyclic.Group, batchSize uint32, roundBuffer *round.Buffer,
	keysPayloadA, cypherPayloadA, keysPayloadB, cypherPayloadB *cyclic.IntBuffer,
	keysPayloadAPermuted, cypherPayloadAPermuted, keysPayloadBPermuted, cypherPayloadBPermuted []*cyclic.Int) {

	ps.Grp = grp
	ps.PublicCypherKey = roundBuffer.CypherPublicKey

	ps.S = roundBuffer.S.GetSubBuffer(0, batchSize)
	ps.V = roundBuffer.V.GetSubBuffer(0, batchSize)
	ps.Y_S = roundBuffer.Y_S.GetSubBuffer(0, batchSize)
	ps.Y_V = roundBuffer.Y_V.GetSubBuffer(0, batchSize)

	ps.KeysPayloadA = keysPayloadA
	ps.CypherPayloadA = cypherPayloadA
	ps.KeysPayloadB = keysPayloadB
	ps.CypherPayloadB = cypherPayloadB

	ps.CypherPayloadBPermuted = cypherPayloadBPermuted
	ps.CypherPayloadAPermuted = cypherPayloadAPermuted

	// these are connected to the round buffer on last node so they are stored
	// during the reveal phase for use in strip
	if len(roundBuffer.PermutedPayloadAKeys) != 0 {
		ps.KeysPayloadAPermuted = roundBuffer.PermutedPayloadAKeys
	} else {
		ps.KeysPayloadAPermuted = keysPayloadAPermuted
	}

	if len(roundBuffer.PermutedPayloadBKeys) != 0 {
		ps.KeysPayloadBPermuted = roundBuffer.PermutedPayloadBKeys
	} else {
		ps.KeysPayloadBPermuted = keysPayloadBPermuted
	}

	graphs.PrecanPermute(roundBuffer.Permutations,
		graphs.PermuteIO{
			Input:  ps.CypherPayloadA,
			Output: ps.CypherPayloadAPermuted,
		}, graphs.PermuteIO{
			Input:  ps.CypherPayloadB,
			Output: ps.CypherPayloadBPermuted,
		}, graphs.PermuteIO{
			Input:  ps.KeysPayloadB,
			Output: ps.KeysPayloadBPermuted,
		}, graphs.PermuteIO{
			Input:  ps.KeysPayloadA,
			Output: ps.KeysPayloadAPermuted,
		},
	)
}

type permuteSubstreamInterface interface {
	GetPrecompPermuteSubStream() *PermuteStream
}

// getSubStream implements reveal interface to return stream object
func (ps *PermuteStream) GetPrecompPermuteSubStream() *PermuteStream {
	return ps
}

// Input initializes stream inputs from slot
func (ps *PermuteStream) Input(index uint32, slot *mixmessages.Slot) error {

	if index >= uint32(ps.KeysPayloadA.Len()) {
		return services.ErrOutsideOfBatch
	}

	if !ps.Grp.BytesInside(slot.EncryptedPayloadAKeys, slot.PartialPayloadACypherText,
		slot.EncryptedPayloadBKeys, slot.PartialPayloadBCypherText) {
		return services.ErrOutsideOfGroup
	}

	ps.Grp.SetBytes(ps.KeysPayloadA.Get(index), slot.EncryptedPayloadAKeys)
	ps.Grp.SetBytes(ps.KeysPayloadB.Get(index), slot.EncryptedPayloadBKeys)
	ps.Grp.SetBytes(ps.CypherPayloadA.Get(index), slot.PartialPayloadACypherText)
	ps.Grp.SetBytes(ps.CypherPayloadB.Get(index), slot.PartialPayloadBCypherText)
	return nil
}

// Output returns a cmix slot message
func (ps *PermuteStream) Output(index uint32) *mixmessages.Slot {
	return &mixmessages.Slot{
		Index:                     index,
		EncryptedPayloadAKeys:     ps.KeysPayloadAPermuted[index].Bytes(),
		EncryptedPayloadBKeys:     ps.KeysPayloadBPermuted[index].Bytes(),
		PartialPayloadACypherText: ps.CypherPayloadAPermuted[index].Bytes(),
		PartialPayloadBCypherText: ps.CypherPayloadBPermuted[index].Bytes(),
	}
}

// PermuteElgamal is a module in precomputation permute implementing cryptops.Elgamal
var PermuteElgamal = services.Module{
	// Multiplies in own Encrypted Keys and Partial Cypher Texts
	Adapt: func(streamInput services.Stream, cryptop cryptops.Cryptop, chunk services.Chunk) error {
		pssi, ok := streamInput.(permuteSubstreamInterface)
		elgamal, ok2 := cryptop.(cryptops.ElGamalPrototype)

		if !ok || !ok2 {
			return services.InvalidTypeAssert
		}

		ps := pssi.GetPrecompPermuteSubStream()

		for i := chunk.Begin(); i < chunk.End(); i++ {
			// Execute elgamal on the keys for the message

			// Eq 11.1: Encrypt the Permuted Internode PayloadA Key under Homomorphic Encryption.
			// Eq 13.17: Then multiply the Permuted Internode PayloadA Key under Homomorphic
			// Encryption into the Partial Encrypted PayloadA Precomputation
			// Eq 11.2: Makes the Partial Cypher Text for the Permuted Internode PayloadA Key
			// Eq 13.21: Multiplies the Partial Cypher Text for the Permuted Internode
			// PayloadA Key into the Round PayloadA Partial Cypher Text

			elgamal(ps.Grp, ps.S.Get(i), ps.Y_S.Get(i), ps.PublicCypherKey, ps.KeysPayloadA.Get(i), ps.CypherPayloadA.Get(i))

			// Execute elgamal on the keys for the Associated Data
			// Eq 11.1: Encrypt the Permuted Internode PayloadB Key under Homomorphic Encryption
			// Eq 13.19: Multiplies the Permuted Internode PayloadB Key under
			// Homomorphic Encryption into the Partial Encrypted PayloadB Precomputation
			// Eq 11.2: Makes the Partial Cypher Text for the Permuted Internode PayloadB Key
			// Eq 13.23 Multiplies the Partial Cypher Text for the Permuted Internode
			// PayloadB Key into the Round PayloadB Partial Cypher Text

			elgamal(ps.Grp, ps.V.Get(i), ps.Y_V.Get(i), ps.PublicCypherKey, ps.KeysPayloadB.Get(i), ps.CypherPayloadB.Get(i))
		}
		return nil
	},
	Cryptop:    cryptops.ElGamal,
	NumThreads: services.AutoNumThreads,
	InputSize:  services.AutoInputSize,
	Name:       "PermuteElgamal",
}

// InitPermuteGraph is called to initialize the graph. Conforms to graphs.Initialize function type
func InitPermuteGraph(gc services.GraphGenerator) *services.Graph {
	gcPermute := graphs.ModifyGraphGeneratorForPermute(gc)
	g := gcPermute.NewGraph("PrecompPermute", &PermuteStream{})

	PermuteElgamal := PermuteElgamal.DeepCopy()

	g.First(PermuteElgamal)
	g.Last(PermuteElgamal)

	return g
}
