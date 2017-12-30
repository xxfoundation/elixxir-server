package globals

import (
	"gitlab.com/privategrity/cyclic/int"
)

// Create and populates the arrays that hold all the keys for a
// Node. Those keys are the R, S, T U,V, their associated private
// keys. Also create structures for the base transmission and reception
// keys.

// A structure also needs to be created to store the Homomorphicly
// Encrypted variants of R, S, T, U, and V.

// A structure which is used to store the results of the Permuted Phase
// in both the real time and the Precomputation is needed so that the
// results of these phases can be outputted as a group to ensure no
// information about the ordering is revealed.

// Also needs to hold permutations

// Keys for Homomorphic operations
var Z *cyclic.Int // Node Cypher key
var G *cyclic.Int // Global Generator

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

// The Rounds map is a mapping of session identifiers to round structurs
var Rounds map[string]round
