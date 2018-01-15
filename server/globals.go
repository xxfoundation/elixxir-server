package server

import (
	"gitlab.com/privategrity/crypto/cyclic"
)

type LastNode struct {
	// Message Decryption key, AKA PiRST_Inv
	MessagePrecomputation []*cyclic.Int
	// Recipient ID Decryption Key, AKA PiUV_Inv
	RecipientPrecomputation []*cyclic.Int
}

// The round struct contains the keys and permutations for a given message batch
type Round struct {
	R            []*cyclic.Int // First unpermuted internode message key
	S            []*cyclic.Int // Permuted internode message key
	T            []*cyclic.Int // Second unpermuted internode message key
	V            []*cyclic.Int // Unpermuted internode recipient key
	U            []*cyclic.Int // Permuted *cyclic.Internode receipient key
	R_INV        []*cyclic.Int // First Inverse unpermuted internode message key
	S_INV        []*cyclic.Int // Permuted Inverse internode message key
	T_INV        []*cyclic.Int // Second Inverse unpermuted internode message key
	V_INV        []*cyclic.Int // Unpermuted Inverse internode recipient key
	U_INV        []*cyclic.Int // Permuted Inverse *cyclic.Internode receipient key
	Permutations []uint64      // Permutation array, messages at index i become
	// messages at index Permutations[i]
	G *cyclic.Int // Global Cypher Key
	Z *cyclic.Int // This node's Cypher Key
	// Private keys for the above
	Y_R []*cyclic.Int
	Y_S []*cyclic.Int
	Y_T []*cyclic.Int
	Y_V []*cyclic.Int
	Y_U []*cyclic.Int

	// Variables only carried by the last node
	LastNode

	BatchSize uint64
}

// Keys for Homomorphic operations
var G *cyclic.Int // Global Generator

//Group that all operations are done within
var Grp *cyclic.Group

// The Rounds map is a mapping of session identifiers to round structures
var Rounds map[string]*Round

var TestArray = [2]float32{.03, .02}

// NewRound constructs an empty round for a given batch size, with all
// numbers being initialized to 0.
func NewRound(batchSize uint64) *Round {
	NR := Round{
		R: make([]*cyclic.Int, batchSize),
		S: make([]*cyclic.Int, batchSize),
		T: make([]*cyclic.Int, batchSize),
		V: make([]*cyclic.Int, batchSize),
		U: make([]*cyclic.Int, batchSize),

		R_INV: make([]*cyclic.Int, batchSize),
		S_INV: make([]*cyclic.Int, batchSize),
		T_INV: make([]*cyclic.Int, batchSize),
		V_INV: make([]*cyclic.Int, batchSize),
		U_INV: make([]*cyclic.Int, batchSize),

		G: cyclic.NewInt(0),
		Z: cyclic.NewInt(0),

		Permutations: make([]uint64, batchSize),

		Y_R: make([]*cyclic.Int, batchSize),
		Y_S: make([]*cyclic.Int, batchSize),
		Y_T: make([]*cyclic.Int, batchSize),
		Y_V: make([]*cyclic.Int, batchSize),
		Y_U: make([]*cyclic.Int, batchSize),

		BatchSize: batchSize}

	NR.G.SetBytes(cyclic.Max4kBitInt)
	NR.Z.SetBytes(cyclic.Max4kBitInt)

	for i := uint64(0); i < batchSize; i++ {
		NR.R[i] = cyclic.NewInt(0)
		NR.S[i] = cyclic.NewInt(0)
		NR.T[i] = cyclic.NewInt(0)
		NR.V[i] = cyclic.NewInt(0)
		NR.U[i] = cyclic.NewInt(0)

		NR.R_INV[i] = cyclic.NewInt(0)
		NR.S_INV[i] = cyclic.NewInt(0)
		NR.T_INV[i] = cyclic.NewInt(0)
		NR.V_INV[i] = cyclic.NewInt(0)
		NR.U_INV[i] = cyclic.NewInt(0)

		NR.Y_R[i] = cyclic.NewInt(0)
		NR.Y_S[i] = cyclic.NewInt(0)
		NR.Y_T[i] = cyclic.NewInt(0)
		NR.Y_V[i] = cyclic.NewInt(0)
		NR.Y_U[i] = cyclic.NewInt(0)

		NR.R[i].SetBytes(cyclic.Max4kBitInt)
		NR.S[i].SetBytes(cyclic.Max4kBitInt)
		NR.T[i].SetBytes(cyclic.Max4kBitInt)
		NR.V[i].SetBytes(cyclic.Max4kBitInt)
		NR.U[i].SetBytes(cyclic.Max4kBitInt)

		NR.R_INV[i].SetBytes(cyclic.Max4kBitInt)
		NR.S_INV[i].SetBytes(cyclic.Max4kBitInt)
		NR.T_INV[i].SetBytes(cyclic.Max4kBitInt)
		NR.V_INV[i].SetBytes(cyclic.Max4kBitInt)
		NR.U_INV[i].SetBytes(cyclic.Max4kBitInt)

		NR.Y_R[i].SetBytes(cyclic.Max4kBitInt)
		NR.Y_S[i].SetBytes(cyclic.Max4kBitInt)
		NR.Y_T[i].SetBytes(cyclic.Max4kBitInt)
		NR.Y_V[i].SetBytes(cyclic.Max4kBitInt)
		NR.Y_U[i].SetBytes(cyclic.Max4kBitInt)

		NR.Permutations[i] = i

		NR.Last = nil
	}

	return &NR
}
