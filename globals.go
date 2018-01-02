package globals

import (
	"gitlab.com/privategrity/cyclic/int"
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
