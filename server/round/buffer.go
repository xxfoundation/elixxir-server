////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package round

import (
	"gitlab.com/elixxir/crypto/cyclic"
)

type Buffer struct {
	batchSize         uint32
	expandedBatchSize uint32

	// BatchWide Keys
	CypherPublicKey *cyclic.Int // Global Cypher Key
	Z               *cyclic.Int // This node's private Cypher Key

	//Realtime Keys
	R *cyclic.IntBuffer // First unpermuted internode message key
	S *cyclic.IntBuffer // Permuted internode message key
	U *cyclic.IntBuffer // Permuted *cyclic.Internode recipient key
	V *cyclic.IntBuffer // Unpermuted internode associated data key

	// Private keys for the above
	Y_R *cyclic.IntBuffer
	Y_S *cyclic.IntBuffer
	Y_T *cyclic.IntBuffer
	Y_V *cyclic.IntBuffer
	Y_U *cyclic.IntBuffer

	// Pre-populated permutations
	Permutations []uint32

	// Results of Precomputation
	MessagePrecomputation *cyclic.IntBuffer
	ADPrecomputation      *cyclic.IntBuffer
}

// Function to initialize a new round
func NewBuffer(g *cyclic.Group, batchSize, expandedBatchSize uint32) *Buffer {

	permutations := make([]uint32, expandedBatchSize)
	for i := uint32(0); i < expandedBatchSize; i++ {
		permutations[i] = i
	}

	return &Buffer{
		R: g.NewIntBuffer(expandedBatchSize, g.NewInt(1)),
		S: g.NewIntBuffer(expandedBatchSize, g.NewInt(1)),
		V: g.NewIntBuffer(expandedBatchSize, g.NewInt(1)),
		U: g.NewIntBuffer(expandedBatchSize, g.NewInt(1)),

		CypherPublicKey: g.NewMaxInt(),
		Z:               g.NewMaxInt(),

		Y_R: g.NewIntBuffer(expandedBatchSize, g.NewInt(1)),
		Y_S: g.NewIntBuffer(expandedBatchSize, g.NewInt(1)),
		Y_T: g.NewIntBuffer(expandedBatchSize, g.NewInt(1)),
		Y_V: g.NewIntBuffer(expandedBatchSize, g.NewInt(1)),
		Y_U: g.NewIntBuffer(expandedBatchSize, g.NewInt(1)),

		Permutations: permutations,

		batchSize:         batchSize,
		expandedBatchSize: expandedBatchSize,

		MessagePrecomputation: g.NewIntBuffer(expandedBatchSize, g.NewInt(1)),
		ADPrecomputation:      g.NewIntBuffer(expandedBatchSize, g.NewInt(1)),
	}
}

func (r *Buffer) GetBatchSize() uint32 {
	return r.batchSize
}

func (r *Buffer) GetExpandedBatchSize() uint32 {
	return r.expandedBatchSize
}
