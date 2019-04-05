package node

import (
	"gitlab.com/elixxir/crypto/cyclic"
)

type RoundID uint64

type Round struct {
	id RoundID

	R    *cyclic.IntBuffer  // First unpermuted internode message key
	S    *cyclic.IntBuffer  // Permuted internode message key
	T    *cyclic.IntBuffer  // Second unpermuted internode message key
	V    *cyclic.IntBuffer  // Unpermuted internode associated data key
	U    *cyclic.IntBuffer  // Permuted *cyclic.Internode recipient key
	Rinv *cyclic.IntBuffer  // First Inverse unpermuted internode message key
	Sinv *cyclic.IntBuffer  // Permuted Inverse internode message key
	Tinv *cyclic.IntBuffer  // Second Inverse unpermuted internode message key
	Vinv *cyclic.IntBuffer  // Unpermuted Inverse internode associated data key
	Uinv *cyclic.IntBuffer  // Permuted Inverse *cyclic.Internode recipient key

	CypherPublicKey *cyclic.Int // Global Cypher Key
	Z               *cyclic.Int // This node's private Cypher Key

	// Private keys for the above
	Y_R *cyclic.IntBuffer
	Y_S *cyclic.IntBuffer
	Y_T *cyclic.IntBuffer
	Y_V *cyclic.IntBuffer
	Y_U *cyclic.IntBuffer

	// Size of batch
	batchSize         uint32
	expandedBatchSize uint32
}

// Function to initialize a new round
func NewRound(g *cyclic.Group, batchsize, expandedBatchSize uint32) *Round {

	return &Round{
		R: g.NewIntBuffer(expandedBatchSize, g.NewInt(1)),
		S: g.NewIntBuffer(expandedBatchSize, g.NewInt(1)),
		T: g.NewIntBuffer(expandedBatchSize, g.NewInt(1)),
		V: g.NewIntBuffer(expandedBatchSize, g.NewInt(1)),
		U: g.NewIntBuffer(expandedBatchSize, g.NewInt(1)),

		Rinv: g.NewIntBuffer(expandedBatchSize, g.NewInt(1)),
		Sinv: g.NewIntBuffer(expandedBatchSize, g.NewInt(1)),
		Tinv: g.NewIntBuffer(expandedBatchSize, g.NewInt(1)),
		Vinv: g.NewIntBuffer(expandedBatchSize, g.NewInt(1)),
		Uinv: g.NewIntBuffer(expandedBatchSize, g.NewInt(1)),

		CypherPublicKey: g.NewMaxInt(),
		Z:               g.NewMaxInt(),

		Y_R: g.NewIntBuffer(expandedBatchSize, g.NewInt(1)),
		Y_S: g.NewIntBuffer(expandedBatchSize, g.NewInt(1)),
		Y_T: g.NewIntBuffer(expandedBatchSize, g.NewInt(1)),
		Y_V: g.NewIntBuffer(expandedBatchSize, g.NewInt(1)),
		Y_U: g.NewIntBuffer(expandedBatchSize, g.NewInt(1)),

		batchSize: batchsize,
		expandedBatchSize: expandedBatchSize,
	}
}

func (r *Round)GetBatchSize()uint32{
	return r.batchSize
}

func (r*Round)GetExpandedBatchSize()uint32{
	return r.expandedBatchSize
}
