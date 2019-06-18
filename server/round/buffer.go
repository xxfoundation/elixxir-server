////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package round

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/shuffle"
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

	// Stores the result of the precomputation permuted phase for the last node
	// To reuse in the Identify phase because the Reveal phase does not use the data
	PermutedMessageKeys []*cyclic.Int
	PermutedADKeys      []*cyclic.Int
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

func (r *Buffer) InitLastNode() {
	r.PermutedMessageKeys = make([]*cyclic.Int, r.expandedBatchSize)
	r.PermutedADKeys = make([]*cyclic.Int, r.expandedBatchSize)
}

func (r *Buffer) InitBatchWideKeys(g *cyclic.Group, z *cyclic.Int) {

	// Set private key using small coprime inverse
	// with 256 bits
	bits := uint32(256)
	r.Z = g.FindSmallCoprimeInverse(z, bits)

	// Permute up to the batch size
	if r.batchSize <= r.expandedBatchSize {
		batchSizePerm := r.Permutations[:r.batchSize]
		shuffle.Shuffle32(&batchSizePerm)
	}

}

func (r *Buffer) GetBatchSize() uint32 {
	return r.batchSize
}

func (r *Buffer) GetExpandedBatchSize() uint32 {
	return r.expandedBatchSize
}

// Erase clears all data contained in the buffer. All elements are set to zero
// and all arrays are set to nil. All underlying released data will be removed
// by the garbage collector.
func (r *Buffer) Erase() {
	r.batchSize = 0
	r.expandedBatchSize = 0

	r.CypherPublicKey.Erase()
	r.Z.Erase()

	r.R.Erase()
	r.S.Erase()
	r.U.Erase()
	r.V.Erase()

	r.Y_R.Erase()
	r.Y_S.Erase()
	r.Y_T.Erase()
	r.Y_V.Erase()
	r.Y_U.Erase()

	r.Permutations = nil

	r.MessagePrecomputation.Erase()
	r.ADPrecomputation.Erase()

	r.PermutedMessageKeys = nil
	r.PermutedADKeys = nil
}
