package globals

import (
	"gitlab.com/privategrity/crypto/cyclic"
)

// The round struct contains the keys and permutations for a given message batch
type round struct {
	R []*cyclic.Int // First unpermuted internode message key
	S []*cyclic.Int // Permuted internode messag key
	T []*cyclic.Int // Second unpermuted internode message key
	V []*cyclic.Int // Unpermuted internode recipient key
	U []*cyclic.Int // Permuted *cyclic.Internode receipient key
	Permutations []uint64 // Permutation array, messages at index i become
	                      // messages at index Permutations[i]
	G *cyclic.Int // Global Cypher Key
	// Private keys for the above
	Y_R []*cyclic.Int
	Y_S []*cyclic.Int
	Y_T []*cyclic.Int
	Y_V []*cyclic.Int
	Y_U []*cyclic.Int
}

// Keys for Homomorphic operations
var Z *cyclic.Int // Node Cypher key
var G *cyclic.Int // Global Generator

// The Rounds map is a mapping of session identifiers to round structures
var Rounds map[string]*round

// NewRound constructs an empty round for a given batch size, with all
// numbers being initialized to 0.
func NewRound(batchSize uint64) *round {
	NR := round{
		R: make([]*cyclic.Int, batchSize),
		S: make([]*cyclic.Int, batchSize),
		T: make([]*cyclic.Int, batchSize),
		V: make([]*cyclic.Int, batchSize),
		U: make([]*cyclic.Int, batchSize),
		G: cyclic.NewInt(0),
		Permutations: make([]uint64, batchSize),
		Y_R: make([]*cyclic.Int, batchSize),
		Y_S: make([]*cyclic.Int, batchSize),
		Y_T: make([]*cyclic.Int, batchSize),
		Y_V: make([]*cyclic.Int, batchSize),
		Y_U: make([]*cyclic.Int, batchSize) }

	for i := uint64(0); i < batchSize; i++ {
		NR.R[i] = cyclic.NewInt(0)
		NR.S[i] = cyclic.NewInt(0)
		NR.T[i] = cyclic.NewInt(0)
		NR.V[i] = cyclic.NewInt(0)
		NR.U[i] = cyclic.NewInt(0)

		NR.Y_R[i] = cyclic.NewInt(0)
		NR.Y_S[i] = cyclic.NewInt(0)
		NR.Y_T[i] = cyclic.NewInt(0)
		NR.Y_V[i] = cyclic.NewInt(0)
		NR.Y_U[i] = cyclic.NewInt(0)
	}

	return &NR
}
