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
// Permute phase permutes the message keys, the associated data keys, and their cypher
// text, while multiplying in its own keys.

// PermuteStream holds data containing keys and inputs used by Permute
type PermuteStream struct {
	Grp             *cyclic.Group
	PublicCypherKey *cyclic.Int

	// Link to round object
	S   *cyclic.IntBuffer // Encrypted Inverse Permuted Internode Message Key
	V   *cyclic.IntBuffer // Encrypted Inverse Permuted Internode AssociatedData Key
	Y_S *cyclic.IntBuffer // Permuted Internode Message Partial Cypher Text
	Y_V *cyclic.IntBuffer // Permuted Internode AssociatedData Partial Cypher Text

	// Unique to stream
	KeysMsg           *cyclic.IntBuffer
	KeysMsgPermuted   []*cyclic.Int
	CypherMsg         *cyclic.IntBuffer
	CypherMsgPermuted []*cyclic.Int
	KeysAD            *cyclic.IntBuffer
	KeysADPermuted    []*cyclic.Int
	CypherAD          *cyclic.IntBuffer
	CypherADPermuted  []*cyclic.Int
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
	keysMsg, cypherMsg, keysAD, cypherAD *cyclic.IntBuffer,
	keysMsgPermuted, cypherMsgPermuted, keysADPermuted, cypherADPermuted []*cyclic.Int) {

	ps.Grp = grp
	ps.PublicCypherKey = roundBuffer.CypherPublicKey

	ps.S = roundBuffer.S.GetSubBuffer(0, batchSize)
	ps.V = roundBuffer.V.GetSubBuffer(0, batchSize)
	ps.Y_S = roundBuffer.Y_S.GetSubBuffer(0, batchSize)
	ps.Y_V = roundBuffer.Y_V.GetSubBuffer(0, batchSize)

	ps.KeysMsg = keysMsg
	ps.CypherMsg = cypherMsg
	ps.KeysAD = keysAD
	ps.CypherAD = cypherAD

	ps.CypherADPermuted = cypherADPermuted
	ps.CypherMsgPermuted = cypherMsgPermuted

	if len(roundBuffer.PermutedMessageKeys) != 0 {
		ps.KeysMsgPermuted = roundBuffer.PermutedMessageKeys
	} else {
		ps.KeysMsgPermuted = keysMsgPermuted
	}

	if len(roundBuffer.PermutedADKeys) != 0 {
		ps.KeysADPermuted = roundBuffer.PermutedADKeys
	} else {
		ps.KeysADPermuted = keysADPermuted
	}

	graphs.PrecanPermute(roundBuffer.Permutations,
		graphs.PermuteIO{
			Input:  ps.CypherMsg,
			Output: ps.CypherMsgPermuted,
		}, graphs.PermuteIO{
			Input:  ps.CypherAD,
			Output: ps.CypherADPermuted,
		}, graphs.PermuteIO{
			Input:  ps.KeysAD,
			Output: ps.KeysADPermuted,
		}, graphs.PermuteIO{
			Input:  ps.KeysMsg,
			Output: ps.KeysMsgPermuted,
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

	if index >= uint32(ps.KeysMsg.Len()) {
		return services.ErrOutsideOfBatch
	}

	if !ps.Grp.BytesInside(slot.EncryptedMessageKeys, slot.PartialMessageCypherText,
		slot.EncryptedAssociatedDataKeys, slot.PartialAssociatedDataCypherText) {
		return services.ErrOutsideOfGroup
	}

	ps.Grp.SetBytes(ps.KeysMsg.Get(index), slot.EncryptedMessageKeys)
	ps.Grp.SetBytes(ps.KeysAD.Get(index), slot.EncryptedAssociatedDataKeys)
	ps.Grp.SetBytes(ps.CypherMsg.Get(index), slot.PartialMessageCypherText)
	ps.Grp.SetBytes(ps.CypherAD.Get(index), slot.PartialAssociatedDataCypherText)
	return nil
}

// Output returns a cmix slot message
func (ps *PermuteStream) Output(index uint32) *mixmessages.Slot {
	return &mixmessages.Slot{
		Index:                           index,
		EncryptedMessageKeys:            ps.KeysMsgPermuted[index].Bytes(),
		EncryptedAssociatedDataKeys:     ps.KeysADPermuted[index].Bytes(),
		PartialMessageCypherText:        ps.CypherMsgPermuted[index].Bytes(),
		PartialAssociatedDataCypherText: ps.CypherADPermuted[index].Bytes(),
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
			// Execute elgamal on the keys for the Message

			// Eq 11.1: Encrypt the Permuted Internode Message Key under Homomorphic Encryption.
			// Eq 13.17: Then multiply the Permuted Internode Message Key under Homomorphic
			// Encryption into the Partial Encrypted Message Precomputation
			// Eq 11.2: Makes the Partial Cypher Text for the Permuted Internode Message Key
			// Eq 13.21: Multiplies the Partial Cypher Text for the Permuted Internode
			// Message Key into the Round Message Partial Cypher Text

			elgamal(ps.Grp, ps.S.Get(i), ps.Y_S.Get(i), ps.PublicCypherKey, ps.KeysMsg.Get(i), ps.CypherMsg.Get(i))

			// Execute elgamal on the keys for the Associated Data
			// Eq 11.1: Encrypt the Permuted Internode AssociatedData Key under Homomorphic Encryption
			// Eq 13.19: Multiplies the Permuted Internode AssociatedData Key under
			// Homomorphic Encryption into the Partial Encrypted AssociatedData Precomputation
			// Eq 11.2: Makes the Partial Cypher Text for the Permuted Internode AssociatedData Key
			// Eq 13.23 Multiplies the Partial Cypher Text for the Permuted Internode
			// AssociatedData Key into the Round AssociatedData Partial Cypher Text

			elgamal(ps.Grp, ps.V.Get(i), ps.Y_V.Get(i), ps.PublicCypherKey, ps.KeysAD.Get(i), ps.CypherAD.Get(i))
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
