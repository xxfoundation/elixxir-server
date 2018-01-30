package node

import (
	"errors"
	"gitlab.com/privategrity/crypto/cyclic"
)

// LastNode contains precomputations held only by the last node
type LastNode struct {
	// Message Decryption key, AKA PiRST_Inv
	MessagePrecomputation []*cyclic.Int
	// Recipient ID Decryption Key, AKA PiUV_Inv
	RecipientPrecomputation []*cyclic.Int
	// Round Message Private Key
	RoundMessagePrivateKey []*cyclic.Int
	// Round Recipient Private Key
	RoundRecipientPrivateKey []*cyclic.Int
}

// Round contains the keys and permutations for a given message batch
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
	CypherPublicKey *cyclic.Int // Global Cypher Key
	Z               *cyclic.Int // This node's Cypher Key
	// Private keys for the above
	Y_R []*cyclic.Int
	Y_S []*cyclic.Int
	Y_T []*cyclic.Int
	Y_V []*cyclic.Int
	Y_U []*cyclic.Int

	// Variables only carried by the last node
	LastNode

	BatchSize uint64

	phase Phase
}

// Grp is the cyclic group that all operations are done within
var Grp *cyclic.Group

// Rounds is a mapping of session identifiers to round structures
var Rounds map[string]*Round

var TestArray = [2]float32{.03, .02}

// NewRound constructs an empty round for a given batch size, with all
// numbers being initialized to 0.
func NewRound(batchSize uint64) *Round {
	return newRound(batchSize, OFF)
}

//Creates a new Round at any phase
func NewRoundWithPhase(batchSize uint64, p Phase) *Round {
	return newRound(batchSize, p)
}

//Creates the lastnode object
func InitLastNode(round *Round) {
	round.LastNode.MessagePrecomputation = make([]*cyclic.Int, round.BatchSize)
	round.LastNode.RecipientPrecomputation = make([]*cyclic.Int, round.BatchSize)
	round.LastNode.RoundMessagePrivateKey = make([]*cyclic.Int, round.BatchSize)
	round.LastNode.RoundRecipientPrivateKey = make([]*cyclic.Int, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		round.LastNode.MessagePrecomputation[i] = cyclic.NewMaxInt()
		round.LastNode.RecipientPrecomputation[i] = cyclic.NewMaxInt()
		round.LastNode.RoundMessagePrivateKey[i] = cyclic.NewMaxInt()
		round.LastNode.RoundRecipientPrivateKey[i] = cyclic.NewMaxInt()
	}
}

// Returns a copy of the current phase
func (round *Round) GetPhase() Phase {
	return round.phase
}

// Increments the phase if the phase can be incremented and was told to
// increment to the correct phase
func (round *Round) IncrementPhase(p Phase) error {
	if (round.phase + 1) != p {
		return errors.New("Invalid Phase Incrementation; Expected: %v, Received: %v", (round.phase+1).String(), p.String())
	}

	if round.phase == DONE {
		return errors.New("Cannot Increment Phase past DONE")
	}

	if round.phase == ERROR {
		return errors.New("Cannot Increment a Phase in ERROR")
	}

	round.phase++

	return nil
}

// Puts the phase into an error state
func (round *Round) Error() {
	round.phase = ERROR
}

// Unexported underlying function to initialize a new round
func newRound(batchSize uint64, p Phase) *Round {
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

		CypherPublicKey: cyclic.NewInt(0),
		Z:               cyclic.NewInt(0),

		Permutations: make([]uint64, batchSize),

		Y_R: make([]*cyclic.Int, batchSize),
		Y_S: make([]*cyclic.Int, batchSize),
		Y_T: make([]*cyclic.Int, batchSize),
		Y_V: make([]*cyclic.Int, batchSize),
		Y_U: make([]*cyclic.Int, batchSize),

		BatchSize: batchSize}

	NR.CypherPublicKey.SetBytes(cyclic.Max4kBitInt)
	NR.Z.SetBytes(cyclic.Max4kBitInt)

	for i := uint64(0); i < batchSize; i++ {
		NR.R[i] = cyclic.NewMaxInt()
		NR.S[i] = cyclic.NewMaxInt()
		NR.T[i] = cyclic.NewMaxInt()
		NR.V[i] = cyclic.NewMaxInt()
		NR.U[i] = cyclic.NewMaxInt()

		NR.R_INV[i] = cyclic.NewMaxInt()
		NR.S_INV[i] = cyclic.NewMaxInt()
		NR.T_INV[i] = cyclic.NewMaxInt()
		NR.V_INV[i] = cyclic.NewMaxInt()
		NR.U_INV[i] = cyclic.NewMaxInt()

		NR.Y_R[i] = cyclic.NewMaxInt()
		NR.Y_S[i] = cyclic.NewMaxInt()
		NR.Y_T[i] = cyclic.NewMaxInt()
		NR.Y_V[i] = cyclic.NewMaxInt()
		NR.Y_U[i] = cyclic.NewMaxInt()

		NR.Permutations[i] = i

		NR.LastNode.MessagePrecomputation = nil
		NR.LastNode.RecipientPrecomputation = nil
	}

	NR.phase = p

	return &NR
}
